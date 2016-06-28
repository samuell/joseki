// Copyright (c) 2016 Thomas Minier. All rights reserved.
// Use of this source code is governed by a MIT License
// license that can be found in the LICENSE file.

package tokens

import (
	"errors"
	"github.com/Callidon/joseki/rdf"
	"math/rand"
	"strconv"
)

// TokenEnd represent a RDF URI
type TokenEnd struct {
	*tokenPosition
}

// NewTokenEnd creates a new TokenEnd
func NewTokenEnd(line, row int) *TokenEnd {
	return &TokenEnd{newTokenPosition(line, row)}
}

// Interpret evaluate the token & produce an action.
// In the case of a TokenEnd, it form a new triple using the nodes in the stack
func (t TokenEnd) Interpret(nodeStack *Stack, prefixes *map[string]string, out chan rdf.Triple) error {
	if nodeStack.Len() > 3 {
		return errors.New("encountered a malformed triple pattern at " + t.position())
	}
	object, objIsNode := nodeStack.Pop().(rdf.Node)
	predicate, predIsNode := nodeStack.Pop().(rdf.Node)
	subject, subjIsNode := nodeStack.Pop().(rdf.Node)
	if !objIsNode || !predIsNode || !subjIsNode {
		return errors.New("expected a Node in stack but doesn't found it")
	}
	out <- rdf.NewTriple(subject, predicate, object)
	return nil
}

// TokenSep represent a Turtle separator
type TokenSep struct {
	value string
	*tokenPosition
}

// NewTokenSep creates a new TokenSep
func NewTokenSep(value string, line int, row int) *TokenSep {
	return &TokenSep{value, newTokenPosition(line, row)}
}

// Interpret evaluate the token & produce an action.
// In the case of a TokenSep, it form a new triple based on the separator, using the nodes in the stack
func (t TokenSep) Interpret(nodeStack *Stack, prefixes *map[string]string, out chan rdf.Triple) error {
	// case of a object separator
	if t.value == "[" {
		if nodeStack.Len() > 2 {
			return errors.New("encountered a malformed triple pattern at " + t.position())
		}
		predicate, predIsNode := nodeStack.Pop().(rdf.Node)
		subject, subjIsNode := nodeStack.Pop().(rdf.Node)
		object := rdf.NewBlankNode("v" + strconv.Itoa(rand.Int()))
		if !predIsNode || !subjIsNode {
			return errors.New("expected a Node in stack but doesn't found it")
		}
		out <- rdf.NewTriple(subject, predicate, object)
		nodeStack.Push(object)
	} else {
		if nodeStack.Len() > 3 {
			return errors.New("encountered a malformed triple pattern at " + t.position())
		}
		object, objIsNode := nodeStack.Pop().(rdf.Node)
		predicate, predIsNode := nodeStack.Pop().(rdf.Node)
		subject, subjIsNode := nodeStack.Pop().(rdf.Node)
		if !objIsNode || !predIsNode || !subjIsNode {
			return errors.New("expected a Node in stack but doesn't found it")
		}
		out <- rdf.NewTriple(subject, predicate, object)

		switch t.value {
		case ";":
			// push back the subject into the stack
			nodeStack.Push(subject)
		case ",":
			// push back the subject & the predicate into the stack
			nodeStack.Push(subject)
			nodeStack.Push(predicate)
		default:
			return errors.New("Unexpected separator token " + t.value + " - at " + t.position())
		}
	}
	return nil
}
