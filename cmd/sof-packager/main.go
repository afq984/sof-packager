package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/afq984/sof-packager/internal"
	"github.com/afq984/sof-packager/pb"
	"github.com/alecthomas/kingpin"
	"google.golang.org/protobuf/encoding/prototext"
)

func do(ctx context.Context, opts *options) error {
	ctxt, err := os.ReadFile(opts.config)
	if err != nil {
		return fmt.Errorf("cannot read config: %s", err)
	}

	var c pb.BuildConfig
	if err := prototext.Unmarshal(ctxt, &c); err != nil {
		return fmt.Errorf("invalid config: %s", err)
	}

	if opts.commit != "" {
		log.Println("overriding commit to", opts.commit)
		c.Commit = opts.commit
	}

	buildDir, err := os.MkdirTemp("", "sof-packager-*")
	if err != nil {
		return err
	}
	if !opts.keepBuildDir {
		defer func() {
			log.Println("cleaning up temporary build directory")
			os.RemoveAll(buildDir)
		}()
	}
	log.Println("building in", buildDir)

	builder := &internal.Builder{
		OutDir: opts.outDir,
	}
	return builder.Build(ctx, filepath.Dir(opts.config), &c, buildDir)
}

type options struct {
	config       string
	commit       string
	outDir       string
	keepBuildDir bool
}

func main() {
	var opts options
	kingpin.Arg("config", "build configuration").Required().StringVar(&opts.config)
	kingpin.Flag("commit", "override the commit set in the config").StringVar(&opts.commit)
	kingpin.Flag("outdir", "output directory").Short('o').StringVar(&opts.outDir)
	kingpin.Flag("keep-build-dir", "do not remove build directory after build").BoolVar(&opts.keepBuildDir)
	kingpin.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := do(ctx, &opts); err != nil {
		log.Fatal(err)
	}
}
