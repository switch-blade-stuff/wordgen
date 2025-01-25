package main

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/switch-blade-stuff/wordgen/pkg/wordgen"
)

var DEFAULT_COUNT = 10
var DEFAULT_SEED = time.Now().UnixMilli()
var DEFAULT_FILE = os.Stdout

type config struct {
	Name        string
	Word        string
	Productions map[string]string
}

func usage(cmd string) {
	fmt.Fprintf(os.Stderr, "Usage: %s CONFIG [-count=UINT] [-seed=UINT] [-output=FILE]", path.Base(cmd))
}

func parseArgs(cmd string, args []string) (cfg config, count int, seed int64, outFile *os.File) {
	// Parse config TOML
	if _, err := toml.DecodeFile(args[0], &cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read config at %s: %e", args[0], err)
	}

	// Parse count, seed, and output from args
	var argCount *int
	var argSeed *int64
	for i := min(4, len(args)); i > 1; i-- {
		argStr := args[i-1]
		if argStr == "-h" {
			usage(cmd)
			os.Exit(0)
		}

		if argCount == nil && strings.HasPrefix(argStr, "-count=") {
			value, err := strconv.Atoi(argStr[7:])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid argument to -count: %e", err)
				usage(cmd)
				os.Exit(1)
			}
			argCount = &value
		}
		if outFile == nil && strings.HasPrefix(argStr, "-output=") {
			file, err := os.OpenFile(argStr[8:], os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid output file path: %e", err)
				usage(cmd)
				os.Exit(1)
			}
			outFile = file
		}
		if argSeed == nil && strings.HasPrefix(argStr, "-seed=") {
			value, err := strconv.Atoi(argStr[6:])
			if err != nil {
				fmt.Fprintf(os.Stderr, "Invalid argument to -seed: %e", err)
				usage(cmd)
				os.Exit(1)
			}
			value64 := int64(value)
			argSeed = &value64
		}
	}
	if argCount == nil {
		count = DEFAULT_COUNT
	} else {
		count = *argCount
	}
	if argSeed == nil {
		seed = DEFAULT_SEED
	} else {
		seed = *argSeed
	}
	if outFile == nil {
		outFile = DEFAULT_FILE
	}

	return
}

func main() {
	// Parse args
	if len(os.Args) < 2 || len(os.Args) > 5 {
		usage(os.Args[0])
		return
	}
	cfg, count, seed, outFile := parseArgs(os.Args[0], os.Args[1:])
	fmt.Printf("Generating %d words of language `%s` using pattern `%s`\n", count, cfg.Name, cfg.Word)

	// Generate productions
	prodGens := make(map[string]wordgen.Generator)
	for ident, pattern := range cfg.Productions {
		gen, err := wordgen.MakePattern([]rune(pattern))
		if err != nil {
			panic(err)
		}
		prodGens[ident] = gen
	}

	// Generate words
	ctx := wordgen.MakeCtx(seed, prodGens)
	for i := 0; i < count; i++ {
		ctx.Builder.Reset()
		pat, err := wordgen.MakePattern([]rune(cfg.Word))

		_, err = pat.Generate(&ctx)
		if err != nil {
			panic(err)
		}

		fmt.Fprintln(outFile, ctx.Builder.String())
	}
}
