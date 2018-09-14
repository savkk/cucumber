package cucumberexpressions

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
)

var INTEGER_REGEXPS = []*regexp.Regexp{
	regexp.MustCompile(`-?\d+`),
	regexp.MustCompile(`\d+`),
}
var FLOAT_REGEXPS = []*regexp.Regexp{
	regexp.MustCompile(`-?\d*\.\d+`),
}
var WORD_REGEXPS = []*regexp.Regexp{
	regexp.MustCompile(`[^\s]+`),
}
var STRING_REGEXPS = []*regexp.Regexp{
	regexp.MustCompile(`"([^"\\]*(\\.[^"\\]*)*)"|'([^'\\]*(\\.[^'\\]*)*)'`),
}

type ParameterTypeRegistry struct {
	parameterTypeByName    map[string]*ParameterType
	parameterTypesByRegexp map[string][]*ParameterType
}

func NewParameterTypeRegistry() *ParameterTypeRegistry {
	result := &ParameterTypeRegistry{
		parameterTypeByName:    map[string]*ParameterType{},
		parameterTypesByRegexp: map[string][]*ParameterType{},
	}
	intParameterType, err := NewParameterType(
		"int",
		INTEGER_REGEXPS,
		"int",
		func(args ...*string) interface{} {
			i, err := strconv.Atoi(*args[0])
			if err != nil {
				panic(err)
			}
			return i
		},
		true,
		true,
	)
	if err != nil {
		panic(err)
	}
	result.DefineParameterType(intParameterType)
	floatParameterType, err := NewParameterType(
		"float",
		FLOAT_REGEXPS,
		"float",
		func(args ...*string) interface{} {
			f, err := strconv.ParseFloat(*args[0], 64)
			if err != nil {
				panic(err)
			}
			return f
		},
		true,
		false,
	)
	if err != nil {
		panic(err)
	}
	result.DefineParameterType(floatParameterType)
	wordParameterType, err := NewParameterType(
		"word",
		WORD_REGEXPS,
		"string",
		func(args ...*string) interface{} {
			return *args[0]
		},
		false,
		false,
	)
	if err != nil {
		panic(err)
	}
	result.DefineParameterType(wordParameterType)
	stringParameterType, err := NewParameterType(
		"string",
		STRING_REGEXPS,
		"string",
		func(args ...*string) interface{} {
			if args[0] == nil && args[1] != nil {
				return *args[1]
			}
			return *args[0]
		},
		true,
		false,
	)
	if err != nil {
		panic(err)
	}
	result.DefineParameterType(stringParameterType)
	return result
}

func (p *ParameterTypeRegistry) ParamaterTypes() []*ParameterType {
	result := make([]*ParameterType, len(p.parameterTypeByName))
	index := 0
	for _, parameterType := range p.parameterTypeByName {
		result[index] = parameterType
		index++
	}
	return result
}

func (p *ParameterTypeRegistry) LookupByTypeName(name string) *ParameterType {
	return p.parameterTypeByName[name]
}

func (p *ParameterTypeRegistry) LookupByRegexp(parameterTypeRegexp string, expressionRegexp string, text string) (*ParameterType, error) {
	parameterTypes, ok := p.parameterTypesByRegexp[parameterTypeRegexp]
	if !ok {
		return nil, nil
	}
	if len(parameterTypes) > 1 && !parameterTypes[0].PreferForRegexpMatch() {
		generatedExpressions := NewCucumberExpressionGenerator(p).GenerateExpressions(text)
		return nil, NewAmbiguousParameterTypeError(parameterTypeRegexp, expressionRegexp, parameterTypes, generatedExpressions)
	}
	return parameterTypes[0], nil
}

func (p *ParameterTypeRegistry) DefineParameterType(parameterType *ParameterType) error {
	if len(parameterType.Name()) > 0 {
		if _, ok := p.parameterTypeByName[parameterType.Name()]; ok {
			return fmt.Errorf("There is already a parameter type with name %s", parameterType.Name())
		}
		p.parameterTypeByName[parameterType.Name()] = parameterType
	}
	for _, parameterTypeRegexp := range parameterType.Regexps() {
		if _, ok := p.parameterTypesByRegexp[parameterTypeRegexp.String()]; !ok {
			p.parameterTypesByRegexp[parameterTypeRegexp.String()] = []*ParameterType{}
		}
		parameterTypes := p.parameterTypesByRegexp[parameterTypeRegexp.String()]
		if len(parameterTypes) > 0 && parameterTypes[0].PreferForRegexpMatch() && parameterType.PreferForRegexpMatch() {
			return fmt.Errorf("There can only be one preferential parameter type per regexp. The regexp /%s/ is used for two preferential parameter types, {%s} and {%s}", parameterTypeRegexp.String(), parameterTypes[0].Name(), parameterType.Name())
		}
		parameterTypes = append(parameterTypes, parameterType)
		sort.Slice(parameterTypes, func(i int, j int) bool {
			return CompareParameterTypes(parameterTypes[i], parameterTypes[j]) <= 0
		})
		p.parameterTypesByRegexp[parameterTypeRegexp.String()] = parameterTypes
	}
	return nil
}
