package wordgen

import (
	"math/rand"
	"strings"
)

type Ctx struct {
	Builder strings.Builder
	rnd     rand.Source
	tbl     map[string]Generator
	// maxSeq  uint
}

func MakeCtx(seed int64, prod map[string]Generator) Ctx {
	return Ctx{
		Builder: strings.Builder{},
		rnd:     rand.NewSource(seed),
		tbl:     prod,
	}
}

type Generator interface {
	Generate(ctx *Ctx) (int, error)
}
