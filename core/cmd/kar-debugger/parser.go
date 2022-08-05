package main

// TODO: this file is an exact duplicate of runtime/debug-parser.go
// restructure the directories!

import (
	"fmt"
	"strconv"
	"strings"
	"regexp"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

func access(obj interface{}, query []string) (interface{}, error) {
	for _, curQuery := range query {
		switch t := obj.(type) {
		case []interface{}:
			// we have an array
			// only valid accessor is an integer index
			idx, err := strconv.Atoi(curQuery)
			if err != nil { return nil,
				fmt.Errorf("Array accessor must be an int literal")
			}
			if idx >= len(t) {
				return nil, fmt.Errorf("Array accessor %v out of bounds for array of length %v", idx, len(t))
			}
			obj = t[idx]
		case map[string]interface{}:
			// interface{} string is a valid identifier now
			// -- as long as the index exists
			newObj, ok := t[curQuery]
			if !ok {
				return nil, fmt.Errorf("Key %v not found in map", curQuery)
			}
			obj = newObj
		default:
			return obj, fmt.Errorf("Attempted to access property %v of a non-map/non-array %v", curQuery, obj)
		}
	}
	return obj, nil
}

// parsing stuff

type CondList_p struct {
	Head *[]*Cond_p `(@@ "," Whitespace?)*`
	Tail Cond_p `@@`
}

type Cond_p struct {
	LHS Expr_p `@@ `
	Op string `Whitespace? @CondOp Whitespace?`
	RHS Expr_p `@@`
}

type Accessor_p struct {
	List []string `( ("." @Ident) | ("[" @Int "]"))+`
}

type Expr_p struct {
	Int *int `@Int |`
	Float *float64 `@Float |`
	String *string `@String |`
	Accessor *Accessor_p `@@`
}

var (
	// below rules partially stolen from https://github.com/alecthomas/participle/blob/master/_examples/sql/main.go
	myLexer = lexer.MustSimple([]lexer.SimpleRule{
		{`CondOp`, `!=|<=|>=|==|[<>]|(\b(LIKE|IN)\b)`},
		{`Ident`, `[a-zA-Z][a-zA-Z0-9_]*`},
		{`Int`, `0|(-?[1-9][0-9]*)`},
		{`Punctuation`, `\.|,`},
		{`Float`, `-?[0-9]+\.[0-9]+`},
		{`String`, `"[^"]*"`},
		{`Brackets`, `\[|\]`},
		{"Whitespace", `\s+`},
	})
	parser = participle.MustBuild(
		&CondList_p{},
		participle.Lexer(myLexer),
		participle.Unquote("String"),
		participle.UseLookahead(1024),
	)
)


func cond(obj interface{}, condStruct Cond_p) bool {
	var lhs, rhs interface{}

	lhsStruct := condStruct.LHS
	if lhsStruct.Int != nil { lhs = *lhsStruct.Int }
	if lhsStruct.String != nil { lhs = *lhsStruct.String }
	if lhsStruct.Float != nil { lhs = *lhsStruct.Float }
	if lhsStruct.Accessor != nil {
		var err error
		lhs, err = access(obj, lhsStruct.Accessor.List)
		if err != nil { return false }
	}

	rhsStruct := condStruct.RHS
	if rhsStruct.Int != nil { rhs = *rhsStruct.Int }
	if rhsStruct.String != nil { rhs = *rhsStruct.String }
	if rhsStruct.Float != nil { rhs = *rhsStruct.Float }
	if rhsStruct.Accessor != nil {
		var err error
		rhs, err = access(obj, rhsStruct.Accessor.List)
		if err != nil { return false }
	}
	
	if lhs == nil || rhs == nil { return false }

	if condStruct.Op == "==" {
		return lhs == rhs
	}
	if condStruct.Op == "!=" {
		return lhs != rhs
	}

	_, lfok := lhs.(float64)
	_, rfok := rhs.(float64)
	li, liok := lhs.(int)
	ri, riok := rhs.(int)
	if (lfok && riok) {
		rhs = float64(ri)
	}
	if (rfok && liok) {
		lhs = float64(li)
	}

	switch lhst := lhs.(type) {
	case int:
		if condStruct.Op == "IN" {
			rhst, ok := rhs.([]int)
			if !ok {
				for _, item := range rhst {
					if item == lhst { return true }
				}
				return false
			}
			return false
		}

		rhst, ok := rhs.(int)
		if !ok { return false }

		switch condStruct.Op {
		case "<=":
			return lhst <= rhst
		case ">=":
			return lhst >= rhst
		case "<":
			//fmt.Printf("%v < %v\n\n", lhst, rhst)
			return lhst < rhst
		case ">":
			return lhst > rhst
		}

	case float64:
		if condStruct.Op == "IN" {
			rhst, ok := rhs.([]float64)
			if !ok {
				for _, item := range rhst {
					if item == lhst { return true }
				}
				return false
			}
			return false
		}

		rhst, ok := rhs.(float64)
		if !ok { return false }

		switch condStruct.Op {
		case "<=":
			return lhst <= rhst
		case ">=":
			return lhst >= rhst
		case "<":
			return lhst < rhst
		case ">":
			return lhst > rhst
		}

	case string:
		switch condStruct.Op {
		case "LIKE":
			rhst, ok := rhs.(string)
			if !ok { return false }

			match, _ := regexp.Match(rhst, []byte(lhst))
			return match
		case "IN":
			rhss, ok := rhs.(string)
			if ok {
				return strings.Contains(rhss, lhst)
			}
			rhsm, ok := rhs.(map[string]interface{})
			if ok {
				_, found := rhsm[lhst]
				return found
			}
			rhsss, ok := rhs.([]string)
			if ok {
				for _, item := range rhsss {
					if item == lhst { return true }
				}
				return false
			}
			return false
		}
	}
	return false
}

func runConds(obj interface{}, conds string) bool {
	//fmt.Printf("running conds: %v, %v\n", obj, conds)
	condListStruct := CondList_p {}
	err := parser.ParseString("", conds, &condListStruct)
	if err != nil { /*fmt.Printf("Parse error: %v\nConds: %v\n", err, conds);*/ return false }
	condList := []Cond_p{}

	if condListStruct.Head != nil {
		for _, c := range *(condListStruct.Head) {
			condList = append(condList, *c)
		}
	}
	condList = append(condList, condListStruct.Tail)
	for _, condStruct := range condList {
		if !cond(obj, condStruct) {
			return false
		}
	}
	return true
}
