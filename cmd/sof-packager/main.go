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

func do(ctx context.Context, config, outDir string, keepBuildDir bool) error {
	ctxt, err := os.ReadFile(config)
	if err != nil {
		return fmt.Errorf("cannot read config: %s", err)
	}

	var c pb.BuildConfig
	if err := prototext.Unmarshal(ctxt, &c); err != nil {
		return fmt.Errorf("invalid config: %s", err)
	}

	buildDir, err := os.MkdirTemp("", "sof-packager-*")
	if err != nil {
		return err
	}
	if !keepBuildDir {
		defer func() {
			log.Println("cleaning up temporary build directory")
			os.RemoveAll(buildDir)
		}()
	}
	log.Println("building in", buildDir)

	builder := &internal.Builder{
		OutDir: outDir,
	}
	return builder.Build(ctx, filepath.Dir(config), &c, buildDir)
}

func main() {
	var (
		config       = kingpin.Arg("config", "build configuration").Required().String()
		outDir       = kingpin.Flag("outdir", "output directory").Short('o').String()
		keepBuildDir = kingpin.Flag("keep-build-dir", "do not remove build directory after build").Bool()
	)
	kingpin.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := do(ctx, *config, *outDir, *keepBuildDir); err != nil {
		log.Fatal(err)
	}
}
