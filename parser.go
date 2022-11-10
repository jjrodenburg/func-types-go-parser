// TODO: switch arg to param naming
package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"strings"
)

const (
	FUNC_NAME_STATE = iota
	ARG_TYPE_STATE
	ARG_NAME_STATE
	OPERATOR_TOKEN_STATE
	INSTANCE_CONTEXT_TOKEN_STATE
	FUNC_TYPE_CONSTRUCTOR_TOKEN_STATE
	CLOSE_OUTPUT_BLOCK_STATE
)

const (
	OPERATOR_TOKEN     = "::"
	INSTANCE_CONTEXT   = "=>"
	FUNC_CONSTRUCTOR   = "->"
	OPEN_OUTPUT_BLOCK  = "("
	CLOSE_OUTPUT_BLOCK = ")"
)

func Parse(funcPrototype string) (ast.FuncDecl, error) {
	funcDecleration := ast.FuncDecl{
		Doc:  nil,
		Recv: nil,
		Name: nil,
		Type: nil,
		Body: nil,
	}

	fieldsBuffer := &ast.FieldList{}
	inOutputBlock := false
	previousTypeDescription := "" // TODO: use previously stored field

	tokens := splitPrototypeIntoTokens(funcPrototype)

	currentState := FUNC_NAME_STATE
	for pos, t := range tokens {
		if err := containsForbiddenCharacter(t); err != nil {
			return ast.FuncDecl{}, err
		}

		switch t {
		case OPERATOR_TOKEN:
			currentState = OPERATOR_TOKEN_STATE
		case INSTANCE_CONTEXT:
			currentState = INSTANCE_CONTEXT_TOKEN_STATE
		case FUNC_CONSTRUCTOR:
			currentState = FUNC_TYPE_CONSTRUCTOR_TOKEN_STATE
		case OPEN_OUTPUT_BLOCK:
			inOutputBlock = true
			continue
		case CLOSE_OUTPUT_BLOCK:
			currentState = CLOSE_OUTPUT_BLOCK_STATE
		}

		switch currentState {
		case FUNC_NAME_STATE:
			funcDecleration.Name = ast.NewIdent(t)
			currentState = OPERATOR_TOKEN_STATE

		case OPERATOR_TOKEN_STATE:
			if t != OPERATOR_TOKEN {
				return ast.FuncDecl{}, fmt.Errorf("expected '::' but found %s", t)
			}

			// if the operator is followed by no parameters, we return an error
			// for example: "TestFunc ::"
			if lookAhead(pos, tokens) == "" {
				return ast.FuncDecl{}, fmt.Errorf("provided operator token but no parameters")
			}

			funcDecleration.Type = &ast.FuncType{
				Func:       token.NoPos,
				TypeParams: nil,
				Params:     nil,
				Results:    nil,
			}

			currentState = ARG_TYPE_STATE

		case INSTANCE_CONTEXT_TOKEN_STATE:
			funcDecleration.Recv.List = append(funcDecleration.Recv.List, &ast.Field{
				Names: []*ast.Ident{{
					Name: t,
				}},
			})

			currentState = ARG_TYPE_STATE

		case ARG_TYPE_STATE:
			nextToken := lookAhead(pos, tokens)

			/// if next token is a blank string or ')', then we are at the end of the prototype
			// if the next token is a '->', then we are actually looking at a name and not a type (so do type inference)
			if strings.TrimSpace(nextToken) == "" || nextToken == FUNC_CONSTRUCTOR || nextToken == CLOSE_OUTPUT_BLOCK {
				previousToken := lookBehind(pos, tokens)
				if previousToken == OPEN_OUTPUT_BLOCK {
					// when checking for type inference, ignore the open output block symbol and look at the pos - 2 for the '->' symbol.
					// for example: "testFunc :: Int a -> (b, c)" -> ignore '(' when checking for type inference on "b"
					previousToken = lookBehind(pos-1, tokens)
				}

				// for example: "TestFunc :: Int a -> Bool" which misses the name of the integer argument
				if previousToken == FUNC_CONSTRUCTOR && previousTypeDescription == "" {
					return ast.FuncDecl{}, fmt.Errorf("typed argument with no name at position %d", pos)
				}

				// The argument does not have a type defined, so try to infer it from the previous typed argument
				// eg. "TestFunc:: Int a -> b" where b becomes an Int.
				if previousTypeDescription == "" {
					return ast.FuncDecl{}, fmt.Errorf("missing type for argument at position %d", pos)
				}

				// don't allow duplicate parameter names
				// for example: "TestFunc :: Int a -> a"
				if inOutputBlock {
					if funcDecleration.Recv != nil {
						for _, param := range funcDecleration.Recv.List {
							for _, name := range param.Names {
								if name.Name == t {
									return ast.FuncDecl{}, fmt.Errorf("duplicate argument name [%s]", t)
								}
							}
						}
					}
				} else {
					if funcDecleration.Type.TypeParams != nil {
						for _, param := range funcDecleration.Type.TypeParams.List {
							for _, name := range param.Names {
								if name.Name == t {
									return ast.FuncDecl{}, fmt.Errorf("duplicate argument name [%s]", t)
								}
							}
						}
					}
				}

				fieldsBuffer.List = append(fieldsBuffer.List, &ast.Field{
					Names: []*ast.Ident{{Name: t}},
					Type: &ast.Ident{
						Name: previousTypeDescription,
					},
				})
			} else {
				fieldsBuffer.List = append(fieldsBuffer.List, &ast.Field{
					Type: &ast.Ident{
						Name: t,
					},
				})

				previousTypeDescription = t
				currentState = ARG_NAME_STATE
			}

		case ARG_NAME_STATE:
			// don't allow duplicate parameter names
			// for example: "TestFunc :: Int a -> a"
			// todo duplicated code
			for _, param := range fieldsBuffer.List {
				for _, name := range param.Names {
					if name.Name == t {
						return ast.FuncDecl{}, fmt.Errorf("duplicate argument name [%s]", t)
					}
				}
			}

			fieldsBuffer.List[len(fieldsBuffer.List)-1].Names = append(fieldsBuffer.List[len(fieldsBuffer.List)-1].Names, &ast.Ident{
				Name: t,
			})

		case FUNC_TYPE_CONSTRUCTOR_TOKEN_STATE:
			if lookAhead(pos, tokens) == "" {
				// this means we have a trailing '->'
				return ast.FuncDecl{}, fmt.Errorf("trailing '->' at position %d", pos)
			}

			// if not in an output block, then add to input args every time we see '->'.
			// an output block can consist of multiple '->' and is handled in the CLOSE_OUTPUT_BLOCK_STATE case.
			if !inOutputBlock {
				if funcDecleration.Type.TypeParams == nil {
					funcDecleration.Type.TypeParams = &ast.FieldList{}
				}

				funcDecleration.Type.TypeParams.List = append(funcDecleration.Type.TypeParams.List, fieldsBuffer.List...)
				fieldsBuffer = &ast.FieldList{}
			}

			currentState = ARG_TYPE_STATE

		case CLOSE_OUTPUT_BLOCK_STATE:
			if !inOutputBlock {
				return ast.FuncDecl{}, errors.New("found closing multiple output block with no open block")
			}

			funcDecleration.Recv = fieldsBuffer
			fieldsBuffer = &ast.FieldList{}

			// if we are not at the end of the slice, we still have tokens after closing the output block.
			// for example: "TestFunc :: Int a -> (a -> b) -> c"
			if pos != len(tokens)-1 {
				return ast.FuncDecl{}, errors.New("invalid content after closing output block")
			}
		}
	}

	if len(fieldsBuffer.List) > 0 {
		// this means we have either:
		// A) one output argument
		// B) invalid syntax containing multiple output arguments without a '->' operator.

		if len(fieldsBuffer.List) == 1 && (len(fieldsBuffer.List[0].Names) == 1 || inOutputBlock) {
			funcDecleration.Recv = fieldsBuffer
		} else {
			return ast.FuncDecl{}, errors.New("invalid output block. you likely have multiple outputs not wrapped in a multi-output block. add open and close params")
		}
	}

	return funcDecleration, nil
}

func splitPrototypeIntoTokens(funcPrototype string) []string {
	if strings.TrimSpace(funcPrototype) == "" {
		return nil
	}

	tokens := []string{}

	spaceSplit := strings.Split(funcPrototype, " ")
	for i := 0; i < len(spaceSplit); i++ {
		if len(spaceSplit[i]) > 1 {
			if string(spaceSplit[i][0]) == OPEN_OUTPUT_BLOCK {
				tokens = append(tokens, OPEN_OUTPUT_BLOCK)
				tokens = append(tokens, spaceSplit[i][1:])
				continue
			}

			if string(spaceSplit[i][len(spaceSplit[i])-1]) == CLOSE_OUTPUT_BLOCK {
				tokens = append(tokens, spaceSplit[i][:len(spaceSplit[i])-1])
				tokens = append(tokens, CLOSE_OUTPUT_BLOCK)
				continue
			}
		}

		tokens = append(tokens, spaceSplit[i])

	}

	return tokens
}

func lookAhead(currentPosition int, funcPrototype []string) string {
	if currentPosition+1 < len(funcPrototype) {
		return funcPrototype[currentPosition+1]
	}

	return ""
}

func lookBehind(currentPosition int, funcPrototype []string) string {
	if currentPosition-1 >= 0 {
		fmt.Printf(funcPrototype[currentPosition-1])
		return funcPrototype[currentPosition-1]
	}

	return ""
}

func containsForbiddenCharacter(input string) error {
	// characters forbidden in both single occurence and word-from
	forbiddenCharacters := []rune{','}

	for _, r := range forbiddenCharacters {
		if strings.ContainsRune(input, r) {
			return fmt.Errorf("argument name [%s] contains invalid character [%s]", input, string(r))
		}
	}

	// some characters are fine on their own, because they are symbols ( '(', ')' ) but are not allowed in tokens not only containing that symbol
	forbiddenPluralCharacters := []rune{'(', ')'}
	if len(input) > 1 {
		for _, r := range forbiddenPluralCharacters {
			if strings.ContainsRune(input, r) {
				return fmt.Errorf("argument name [%s] contains invalid character [%s]", input, string(r))
			}
		}
	}

	return nil
}
