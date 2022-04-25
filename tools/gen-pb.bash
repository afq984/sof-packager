#!/bin/bash
protoc --go_out=. --go_opt=module=github.com/afq984/sof-packager build_config.proto
