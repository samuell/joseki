// Copyright (c) 2016 Thomas Minier. All rights reserved.
// Use of this source code is governed by a MIT License
// license that can be found in the LICENSE file.

package sparql

import (
	"github.com/Callidon/joseki/graph"
	"github.com/Callidon/joseki/rdf"
	"sort"
)

// tripleNode is the lowest level of SPARQL query execution plan.
// Its role is to retrieve bindings according to a triple pattern from a graph.
type tripleNode struct {
	pattern       rdf.Triple
	graph         graph.Graph
	limit, offset int
}

// newTripleNode creates a new tripleNode.
func newTripleNode(pattern rdf.Triple, graph graph.Graph, limit int, offset int) *tripleNode {
	return &tripleNode{pattern, graph, limit, offset}
}

// execute retrieves bindings from a graph that match a triple pattern.
func (n tripleNode) execute() <-chan rdf.BindingsGroup {
	out := make(chan rdf.BindingsGroup, bufferSize)
	// find free vars in triple pattern
	subject, freeSubject := n.pattern.Subject.(rdf.Variable)
	predicate, freePredicate := n.pattern.Predicate.(rdf.Variable)
	object, freeObject := n.pattern.Object.(rdf.Variable)

	// retrieves triples & form bindings to send
	go func() {
		defer close(out)
		for triple := range n.graph.FilterSubset(n.pattern.Subject, n.pattern.Predicate, n.pattern.Object, n.limit, n.offset) {
			group := rdf.NewBindingsGroup()
			if freeSubject {
				group.Bindings[subject.Value] = triple.Subject
			}
			if freePredicate {
				group.Bindings[predicate.Value] = triple.Predicate
			}
			if freeObject {
				group.Bindings[object.Value] = triple.Object
			}
			out <- group
		}
	}()
	return out
}

// executeWith retrieves bindings from a graph that match a triple pattern, completed by a given binding.
func (n tripleNode) executeWith(group rdf.BindingsGroup) <-chan rdf.BindingsGroup {
	var querySubj, queryPred, queryObj rdf.Node
	out := make(chan rdf.BindingsGroup, bufferSize)
	// find free vars in triple pattern
	subject, freeSubject := n.pattern.Subject.(rdf.Variable)
	predicate, freePredicate := n.pattern.Predicate.(rdf.Variable)
	object, freeObject := n.pattern.Object.(rdf.Variable)

	// complete triple pattern using the group of bindings given in parameter
	for bindingKey, bindingValue := range group.Bindings {
		if freeSubject && subject.Value == bindingKey {
			querySubj = bindingValue
			freeSubject = false
		} else {
			querySubj = n.pattern.Subject
		}
		if freePredicate && predicate.Value == bindingKey {
			queryPred = bindingValue
			freePredicate = false
		} else {
			queryPred = n.pattern.Predicate
		}
		if freeObject && object.Value == bindingKey {
			queryObj = bindingValue
			freeObject = false
		} else {
			queryObj = n.pattern.Object
		}
	}

	// retrieves triples & form bindings to send
	go func() {
		defer close(out)
		for triple := range n.graph.FilterSubset(querySubj, queryPred, queryObj, n.limit, n.offset) {
			newGroup := group.Clone()
			if freeSubject {
				newGroup.Bindings[subject.Value] = triple.Subject
			}
			if freePredicate {
				newGroup.Bindings[predicate.Value] = triple.Predicate
			}
			if freeObject {
				newGroup.Bindings[object.Value] = triple.Object
			}
			out <- newGroup
		}
	}()
	return out
}

// bindingNames returns the names of the bindings produced.
func (n tripleNode) bindingNames() (bindingNames []string) {
	// find free vars in triple pattern
	subject, freeSubject := n.pattern.Subject.(rdf.Variable)
	predicate, freePredicate := n.pattern.Predicate.(rdf.Variable)
	object, freeObject := n.pattern.Object.(rdf.Variable)
	if freeSubject {
		bindingNames = append(bindingNames, subject.Value)
	}
	if freePredicate {
		bindingNames = append(bindingNames, predicate.Value)
	}
	if freeObject {
		bindingNames = append(bindingNames, object.Value)
	}
	sort.Strings(bindingNames)
	return
}

// Equals test if two Triple nodes are equals.
func (n tripleNode) Equals(other sparqlNode) bool {
	tripleN, isTriple := other.(*tripleNode)
	if !isTriple {
		return false
	}
	test, _ := n.pattern.Equals(tripleN.pattern)
	return test
}

// String serialize the node in string format.
func (n tripleNode) String() string {
	return "Triple(" + n.pattern.Subject.String() + " " + n.pattern.Predicate.String() + " " + n.pattern.Object.String() + ")"
}
