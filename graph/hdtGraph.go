package graph

import (
	"github.com/Callidon/joseki/rdf"
	"sync"
)

// HDTGraph is a implementation of a RDF Graph based on the HDT-MR model proposed by Giménez-García et al.
//
// For more details, see http://dataweb.infor.uva.es/projects/hdt-mr/
type HDTGraph struct {
	dictionnary bimap
	root        bitmapNode
	nextID      int
	triples     map[string][]rdf.Triple
	*sync.Mutex
	*rdfReader
}

// NewHDTGraph creates a new empty HDT Graph.
func NewHDTGraph() *HDTGraph {
	reader := newRDFReader()
	g := &HDTGraph{newBimap(), newBitmapNode(-1), 0, make(map[string][]rdf.Triple), &sync.Mutex{}, reader}
	reader.graph = g
	return g
}

// Register a new Node in the graph dictionnary, then return its unique ID.
func (g *HDTGraph) registerNode(node rdf.Node) int {
	// insert the node in dictionnary only if it's not in
	key, inDict := g.dictionnary.locate(node)
	if !inDict {
		g.dictionnary.push(g.nextID, node)
		g.nextID++
		return g.nextID - 1
	}
	return key
}

// Recursively update the nodes of the graph with new datas.
func (g *HDTGraph) updateNodes(root *bitmapNode, datas []int) {
	// if they are data to insert in the graph
	if len(datas) > 0 {
		id := datas[0]
		// if the node's id in already in the root sons, continue the operation with it
		node, inSons := root.sons[id]
		if inSons {
			g.updateNodes(node, datas[1:])
		} else {
			// add the new node, then continue the operation with its sons
			root.addSon(id)
			g.updateNodes(root.sons[id], datas[1:])
		}
	}
}

// Recursively remove nodes that match criteria
func (g *HDTGraph) removeNodes(root *bitmapNode, datas []*rdf.Node) {
	// it's a blank node, delete all his sons
	node := (*datas[0])
	if _, isBnode := node.(rdf.BlankNode); isBnode {
		root.removeSons()
	} else {
		// search for the specific node in the root's sons
		refNodeID, inDict := g.dictionnary.locate(node)
		if inDict {
			son, inSons := root.sons[refNodeID]
			if inSons {
				// delete his sons that match the next criteria
				g.removeNodes(son, datas[1:])
			}
		}
	}
}

// Recursively collect datas from the graph in order to form triple pattern matching criterias.
func (g *HDTGraph) queryNodes(root *bitmapNode, datas []*rdf.Node, triple []int, out chan rdf.Triple, wg *sync.WaitGroup) {
	defer wg.Done()
	// when possible, create a new triple pattern & send it into the output pipeline
	if len(triple) == 3 {
		bitmapTriple := newBitmapTriple(triple[0], triple[1], triple[2])
		triple, err := bitmapTriple.Triple(&g.dictionnary)
		if err != nil {
			panic(err)
		}
		out <- triple
	} else {
		node := (*datas[0])
		// if the current node to search is a blank node, search in every sons
		_, isBnode := node.(rdf.BlankNode)
		if isBnode {
			go func() {
				for _, son := range root.sons {
					g.queryNodes(son, datas[1:], append(triple, son.id), out, wg)
				}
			}()
		} else {
			// search for a specific node
			id, inDict := g.dictionnary.locate(node)
			if _, inSons := root.sons[id]; inDict && (inSons || root.sons[id] == nil) {
				go g.queryNodes(root.sons[id], datas[1:], append(triple, id), out, wg)
			}
			// update the counter for the sons that will not be visited
			for key, son := range root.sons {
				if key != id {
					son.updateCounter(wg)
				}
			}
		}
	}
}

// Add a new Triple pattern to the graph.
func (g *HDTGraph) Add(triple rdf.Triple) {
	// add each node of the triple to the dictionnary & then update the graph
	subjID, predID, objID := g.registerNode(triple.Subject), g.registerNode(triple.Predicate), g.registerNode(triple.Object)
	g.Lock()
	defer g.Unlock()
	g.updateNodes(&g.root, []int{subjID, predID, objID})
}

// Delete triples from the graph that match a BGP given in parameters.
func (g *HDTGraph) Delete(subject, object, predicate rdf.Node) {
	g.Lock()
	defer g.Unlock()
	g.removeNodes(&g.root, []*rdf.Node{&subject, &predicate, &object})
}

// Filter fetch triples form the graph that match a BGP given in parameters.
func (g *HDTGraph) Filter(subject, predicate, object rdf.Node) chan rdf.Triple {
	var wg sync.WaitGroup
	results := make(chan rdf.Triple)
	// fetch data in the tree & wait for the operation to be complete before closing the pipeline
	g.Lock()
	wg.Add(g.root.depth() + 1)
	go g.queryNodes(&g.root, []*rdf.Node{&subject, &predicate, &object}, make([]int, 0), results, &wg)
	go func() {
		defer close(results)
		defer g.Unlock()
		wg.Wait()
	}()
	return results
}

// Serialize the graph into a given format and return it as a string.
func (g *HDTGraph) Serialize(format string) string {
	// TODO
	return ""
}
