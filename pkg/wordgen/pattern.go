package wordgen

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"unicode"
)

func times(ctx *Ctx, rngMin uint, rngMax uint, f func() (int, error)) (int, error) {
	var lastErr error = nil
	var currLen int = 0
	var times uint = 0

	// rngMax = min(rngMax, ctx.maxSeq)
	if rngMin != rngMax {
		ratio := float64(ctx.rnd.Int63()) / float64(math.MaxInt64)
		times = uint(math.Round(ratio * float64(rngMax-rngMin)))
		times = min(max(times+rngMin, rngMin), rngMax)
	} else {
		times = rngMin
	}

	for i := uint(0); i < times; i++ {
		len, err := f()
		currLen += len
		lastErr = err

		if lastErr != nil {
			break
		}
	}
	return currLen, lastErr
}

type Pattern struct {
	src string
	gen Generator
}

func (p Pattern) Generate(ctx *Ctx) (int, error) {
	return p.gen.Generate(ctx)
}

type Particle struct {
	min uint
	max uint
	val rune
}

func (p Particle) Generate(ctx *Ctx) (int, error) {
	return times(ctx, p.min, p.max, func() (int, error) { return ctx.Builder.WriteRune(p.val) })
}

type Production struct {
	min  uint
	max  uint
	name string
	prod Generator
}

func (p Production) Generate(ctx *Ctx) (int, error) {
	return times(ctx, p.min, p.max, func() (int, error) {
		if p.prod == nil {
			if prod, ok := ctx.tbl[p.name]; !ok {
				return 0, errors.New(fmt.Sprintf("Unknown production: %s", p.name))
			} else {
				p.prod = prod
			}
		}
		return p.prod.Generate(ctx)
	})
}

type AltGroup struct {
	min uint
	max uint
	alt []Generator
}

func (g AltGroup) Generate(ctx *Ctx) (int, error) {
	return times(ctx, g.min, g.max, func() (int, error) {
		idx := uint(ctx.rnd.Int63()) % uint(len(g.alt))
		return g.alt[idx].Generate(ctx)
	})
}

type SeqGroup struct {
	min uint
	max uint
	seq []Generator
}

func (g SeqGroup) Generate(ctx *Ctx) (int, error) {
	return times(ctx, g.min, g.max, func() (int, error) {
		var lastErr error = nil
		var currLen int = 0
		for _, gen := range g.seq {
			len, err := gen.Generate(ctx)
			currLen += len
			lastErr = err

			if lastErr != nil {
				break
			}
		}
		return currLen, lastErr
	})
}

var eofError = errors.New("Unexpected end of pattern")
var syntaxError = errors.New("Invalid pattern syntax")

const EOF = rune(-1)

type scanner struct {
	src []rune
	pos uint
}

func (s *scanner) peek() rune {
	if s.pos >= uint(len(s.src)) {
		return EOF
	} else {
		return s.src[s.pos]
	}
}
func (s *scanner) next() rune {
	if s.pos >= uint(len(s.src)) {
		return EOF
	} else {
		ch := s.src[s.pos]
		s.pos++
		return ch
	}
}

func (s *scanner) scanInteger() (uint, error) {
	var str []rune
	var err error
	for {
		if ch := s.peek(); unicode.IsNumber(ch) {
			s.next()
			str = append(str, ch)
		} else {
			break
		}
	}

	var val int
	if len(str) > 0 {
		val, err = strconv.Atoi(string(str))
	} else {
		val = 0
		err = errors.New("Invalid number")
	}
	return uint(val), err
}
func (s *scanner) scanIdent() (string, error) {
	var val []rune
	var err error
	if ch := s.peek(); unicode.IsLetter(ch) || ch == '_' {
		s.next()
		val = append(val, ch)

		for {
			ch = s.peek()
			if !(unicode.IsLetter(ch) || unicode.IsNumber(ch) || ch == '_') {
				break
			}
			s.next()
			val = append(val, ch)
		}
	} else {
		err = errors.New("Invalid identifier")
	}

	return string(val), err
}

func (s *scanner) scanRange() (rngMin uint, rngMax uint, err error) {
	rngMin = 1
	rngMax = 1
	err = nil

	switch ch := s.peek(); ch {
	case '?':
		s.next()
		rngMin = 0
	case '{':
		s.next()
		if rngMin, err = s.scanInteger(); err != nil {
			break
		}
		if ch = s.next(); ch != ',' {
			err = errors.New("Expected ','")
			break
		}
		if rngMax, err = s.scanInteger(); err != nil {
			break
		}
		if ch = s.next(); ch != '}' {
			err = errors.New("Expected '}'")
			break
		}
	}
	return
}

func (s *scanner) scanProduction() (Generator, error) {
	if ch := s.peek(); ch == EOF {
		return nil, eofError
	}

	// Scan next ident
	if name, err := s.scanIdent(); err == nil {
		rngMin, rngMax, err := s.scanRange()
		return Production{rngMin, rngMax, name, nil}, err
	} else {
		return nil, err
	}
}

func (s *scanner) scanAltGroup() (Generator, error) {
	var alt []Generator
	for {
		if ch := s.peek(); ch == ']' {
			s.next()
			break
		}
		if gen, err := s.scanNext(false); err != nil {
			return nil, err
		} else {
			alt = append(alt, gen)
		}
	}

	min, max, _ := s.scanRange()
	return AltGroup{min, max, alt}, nil
}

func (s *scanner) scanSeqGroup() (Generator, error) {
	var seq []Generator
	for {
		if ch := s.peek(); ch == ')' {
			s.next()
			break
		}
		if gen, err := s.scanNext(false); err != nil {
			return nil, err
		} else {
			seq = append(seq, gen)
		}
	}

	min, max, _ := s.scanRange()
	return SeqGroup{min, max, seq}, nil
}

func (s *scanner) scanNext(allowEof bool) (Generator, error) {
retry:
	switch ch := s.next(); ch {
	case '\t', ' ':
		goto retry
	case '\\':
		allowEof = false
		s.next()
		goto retry

		// Groups
	case '[':
		return s.scanAltGroup()
	case '(':
		return s.scanSeqGroup()

		// Other
	case '$':
		return s.scanProduction()
	case EOF:
		if !allowEof {
			return nil, eofError
		} else {
			return nil, nil
		}
	default:
		rngMin, rngMax, err := s.scanRange()
		return Particle{rngMin, rngMax, ch}, err
	}
}

func MakePattern(src []rune) (Pattern, error) {
	sc := scanner{src, 0}

	var seq []Generator
	var err error
	for {
		var gen Generator
		if gen, err = sc.scanNext(true); err != nil || gen == nil {
			break
		} else {
			seq = append(seq, gen)
		}
	}
	return Pattern{src: string(src), gen: SeqGroup{1, 1, seq}}, err
}
