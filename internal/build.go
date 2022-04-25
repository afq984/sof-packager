package internal

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/afq984/sof-packager/pb"
	"golang.org/x/crypto/blake2b"
)

type Builder struct {
	OutDir string
}

const (
	dockerRun      = "./scripts/docker-run.sh"
	xtensaBuildAll = "./scripts/xtensa-build-all.sh"
	buildTools     = "./scripts/build-tools.sh"
)

func (b *Builder) buildFirmware(ctx context.Context, fw *pb.Firmware, sof string) error {
	log.Println("building firmware")
	var cmd *exec.Cmd
	if fw.Docker != nil && fw.Docker.Use {
		if err := b.setupDocker(ctx, fw.Docker); err != nil {
			return fmt.Errorf("docker setup failed: %s", err)
		}
		cmd = exec.CommandContext(ctx, dockerRun, xtensaBuildAll)
	} else {
		cmd = exec.CommandContext(ctx, xtensaBuildAll)
	}
	cmd.Dir = sof
	cmd.Args = append(cmd.Args, fw.BuildArg...)

	if err := runCommand(cmd); err != nil {
		return err
	}

	log.Println("firmware build successful")
	return nil
}

func (b *Builder) buildTopology(ctx context.Context, tplg *pb.Topology, sof string) error {
	log.Println("building topology")
	var cmd *exec.Cmd
	if tplg.Docker != nil && tplg.Docker.Use {
		if err := b.setupDocker(ctx, tplg.Docker); err != nil {
			return fmt.Errorf("docker setup failed: %s", err)
		}
		cmd = exec.CommandContext(ctx, dockerRun, buildTools, "-T")
	} else {
		cmd = exec.CommandContext(ctx, buildTools, "-T")
	}
	cmd.Dir = sof

	if err := runCommand(cmd); err != nil {
		return err
	}

	log.Println("topology build successful")
	return nil
}

func (b *Builder) setupDocker(ctx context.Context, docker *pb.DockerConfig) error {
	identifier := docker.Identifier
	if identifier == "" {
		identifier = "thesofproject/sof:latest"
	}

	digest, err := dockerPullAndTag(ctx, identifier, "thesofproject/sof")
	if err != nil {
		return err
	}

	docker.Identifier = digest
	return nil
}

func (b *Builder) Build(ctx context.Context, basedir string, c *pb.BuildConfig, builddir string) error {
	sof := filepath.Join(builddir, "sof")

	if c.Branch == "" {
		c.Branch = "main"
	}

	cmd := exec.CommandContext(ctx,
		"git", "clone", c.Repo, "-b", c.Branch, sof,
	)
	if err := runCommand(cmd); err != nil {
		return fmt.Errorf("git clone failed: %s", err)
	}

	if c.Commit != "" {
		cmd := exec.CommandContext(ctx,
			"git", "checkout", c.Commit,
		)
		cmd.Dir = sof
		if err := runCommand(cmd); err != nil {
			return fmt.Errorf("git checkout failed: %s", err)
		}
	}

	cmd = exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = sof
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("git rev-parse failed: %s", err)
	}
	commit := string(bytes.TrimSpace(out))
	log.Println("using commit", commit)
	c.Commit = commit

	for _, blob := range c.ExtraBlob {
		src := filepath.Join(basedir, blob.Src)
		dst := filepath.Join(sof, blob.Dst)
		log.Println("installing", dst)

		h, err := sha256sum(src)
		if err != nil {
			return fmt.Errorf("cannot compute checksum of %s: %s", src, err)
		}
		if blob.Sha256 != "" && blob.Sha256 != h {
			return fmt.Errorf("mismatched checksum of %s: got: %s; want %s", src, h, blob.Sha256)
		}
		blob.Sha256 = h

		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("failed to copy %s to %s: %s", src, dst, err)
		}
	}

	if c.Firmware != nil {
		if err := b.buildFirmware(ctx, c.Firmware, sof); err != nil {
			return fmt.Errorf("firmware build failed: %s", err)
		}
	}

	if c.Topology != nil {
		if err := b.buildTopology(ctx, c.Topology, sof); err != nil {
			return fmt.Errorf("topology build failed: %s", err)
		}
	}

	if err := b.packageTarball(ctx, c, sof); err != nil {
		return fmt.Errorf("error packaging tarball: %s", err)
	}

	return nil
}

func (b *Builder) packageTarball(ctx context.Context, c *pb.BuildConfig, sof string) error {
	var versionedName string
	if c.Version == "" {
		versionedName = filepath.Join(b.OutDir, c.Tarball)
	} else {
		versionedName = fmt.Sprintf("%s-%s", c.Tarball, c.Version)
	}

	if b.OutDir != "" {
		if err := os.MkdirAll(b.OutDir, 0755); err != nil {
			return err
		}
		versionedName = filepath.Join(b.OutDir, versionedName)
	}

	tarPath := versionedName + ".tar.gz"
	w, err := os.Create(tarPath)
	if err != nil {
		return err
	}
	zw := gzip.NewWriter(w)
	defer zw.Close()
	tb := tar.NewWriter(zw)
	defer tb.Close()

	for _, artifact := range c.Artifact {
		src := filepath.Join(sof, artifact.BuiltPath)

		r, err := os.Open(src)
		if err != nil {
			return err
		}
		defer r.Close()

		st, err := r.Stat()
		if err != nil {
			return fmt.Errorf("failed to stat %s: %s", src, err)
		}

		if artifact.Name == "" {
			artifact.Name = filepath.Base(artifact.BuiltPath)
		}
		nameInTar := artifact.Name
		if !c.FlatTarball {
			nameInTar = path.Join(versionedName, nameInTar)
		}

		if err := tb.WriteHeader(&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     nameInTar,
			Size:     st.Size(),
		}); err != nil {
			return fmt.Errorf("failed to write tar header: %s", err)
		}

		h := sha256.New()
		w := io.MultiWriter(h, tb)

		if _, err := io.Copy(w, r); err != nil {
			return fmt.Errorf("failed to copy file %s: %s", src, err)
		}

		hexdigest := hex.EncodeToString(h.Sum(nil))
		if artifact.Sha256 != "" && artifact.Sha256 != hexdigest {
			return fmt.Errorf("mismatched checksum of %s: built %s; expected %s",
				artifact.BuiltPath, hexdigest, artifact.Sha256)
		}
		artifact.Sha256 = hexdigest

		r.Close()
	}

	// Close
	if err := tb.Close(); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	log.Println("written tarball to", tarPath)

	return b.writeMetadata(ctx, c, tarPath, versionedName)
}

func (b *Builder) writeMetadata(ctx context.Context, c *pb.BuildConfig, tarball, versionedName string) error {
	bytes, err := textproto(c)
	if err != nil {
		panic(err)
	}

	configName := versionedName + ".config.textproto"
	if err := os.WriteFile(configName, bytes, 0644); err != nil {
		return fmt.Errorf("error writing config: %s", err)
	}
	log.Println("written reproducible config to", configName)

	sha512Hash := sha512.New()
	blake2bHash, err := blake2b.New512(nil)
	if err != nil {
		panic(err)
	}
	w := io.MultiWriter(sha512Hash, blake2bHash)
	r, err := os.Open(tarball)
	if err != nil {
		return err
	}
	n, err := io.Copy(w, r)
	if err != nil {
		return err
	}
	ebuildManifest := fmt.Sprintf("DIST %s %d BLAKE2B %x SHA512 %x\n",
		filepath.Base(tarball), n,
		blake2bHash.Sum(nil), sha512Hash.Sum(nil),
	)
	manifestName := versionedName + ".Manifest"
	if err != os.WriteFile(manifestName, []byte(ebuildManifest), 0644) {
		return fmt.Errorf("error writing manifest: %s", err)
	}
	log.Println("written manifest to", manifestName)

	return nil
}
