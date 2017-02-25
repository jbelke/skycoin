package nodemanager

import (
	"fmt"
	"math"

	"github.com/skycoin/skycoin/src/cipher"
	"github.com/skycoin/skycoin/src/mesh/errors"
)

type RouteGraph struct {
	nodes map[cipher.PubKey]map[cipher.PubKey]*DirectRoute
	paths map[cipher.PubKey]*SP
}

type DirectRoute struct {
	from   cipher.PubKey
	to     cipher.PubKey
	weight int
}

func newGraph() *RouteGraph {
	graph := RouteGraph{}
	graph.clear()
	return &graph
}

func (s *RouteGraph) toString() {
	for from, routes := range s.nodes {
		fmt.Println("\nNODE: ", from)
		for _, directRoute := range routes {
			fmt.Println("\tRoute: ", directRoute)
		}
		fmt.Println("======================\n")
	}
}

func (s *RouteGraph) addDirectRoute(from, to cipher.PubKey, weight int) {
	if from == to || weight < 1 {
		return
	}
	if _, ok := s.nodes[from]; !ok {
		s.nodes[from] = map[cipher.PubKey]*DirectRoute{}
	} else {
		if _, ok = s.nodes[from][to]; ok {
			return
		}
	}
	newDirectRoute := &DirectRoute{from, to, weight}
	s.nodes[from][to] = newDirectRoute
}

/* ---- can be useful in the future, maybe
func (s *RouteGraph) RebuildRoutes() {
	s.clear()
	for node := range(nodes) {
		paths[node] = newSP(s, node)
	}
}
*/
func (s *RouteGraph) findRoute(from, to cipher.PubKey) ([]cipher.PubKey, error) {
	sp, found := s.paths[from]
	if !found {
		sp = newSP(s, from)
		s.paths[from] = sp
	}
	route, err := sp.pathTo(to)
	return route, err
}

func (s *RouteGraph) clear() {
	s.nodes = map[cipher.PubKey]map[cipher.PubKey]*DirectRoute{}
	s.paths = map[cipher.PubKey]*SP{}
}

type SP struct { //ShortestPath
	source cipher.PubKey
	distTo map[cipher.PubKey]int
	edgeTo map[cipher.PubKey]*DirectRoute
	pq     *MinPQ
}

func newSP(graph *RouteGraph, source cipher.PubKey) *SP {

	sp := SP{}

	sp.source = source
	sp.distTo = map[cipher.PubKey]int{}
	sp.edgeTo = map[cipher.PubKey]*DirectRoute{}

	for node := range graph.nodes {
		sp.distTo[node] = math.MaxInt32
	}
	sp.distTo[source] = 0

	sp.pq = newPQ()
	sp.pq.insert(source, 0)
	for !sp.pq.isEmpty() {
		v := sp.pq.delMin()
		for _, directRoute := range graph.nodes[v] {
			sp.relax(directRoute)
		}
	}

	return &sp
}

func (s *SP) pathTo(to cipher.PubKey) ([]cipher.PubKey, error) { // if the path exists return a path and true, otherwise empty path and false

	path := []cipher.PubKey{to}
	e := s.edgeTo[to]

	for {
		if e == nil {
			return []cipher.PubKey{}, errors.ERR_NO_ROUTE
		} // no edge, so path doesn't exist
		path = append(path, e.from)
		if e.from == s.source {
			break
		} // we are at the source, work is finished
		e = s.edgeTo[e.from]
	}

	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 { // reverse an slice, in the future should apply stack instead of it
		path[i], path[j] = path[j], path[i]
	}

	return path, nil
}

func (s *SP) relax(edge *DirectRoute) {
	from := edge.from
	to := edge.to
	newDist := s.distTo[from] + edge.weight
	if s.distTo[to] > newDist {
		s.distTo[to] = newDist
		s.edgeTo[to] = edge
		if s.pq.contains(to) {
			s.pq.decreaseKey(to, s.distTo[to])
		} else {
			s.pq.insert(to, s.distTo[to])
		}
	}
}

type NodeDist struct {
	node cipher.PubKey
	dist int
}

type MinPQ struct {
	keys      []*NodeDist
	positions map[cipher.PubKey]int
}

func newPQ() *MinPQ {
	pq := MinPQ{}
	zeroND := &NodeDist{}
	pq.keys = []*NodeDist{zeroND}
	pq.positions = map[cipher.PubKey]int{}
	return &pq
}

func (pq *MinPQ) isEmpty() bool {
	return len(pq.keys) == 1
}

func (pq *MinPQ) contains(node cipher.PubKey) bool {
	_, exists := pq.positions[node]
	return exists
}

func (pq *MinPQ) delMin() cipher.PubKey {
	n := len(pq.keys)
	if pq.isEmpty() {
		return pq.keys[0].node
	}
	min := pq.keys[1].node
	n--
	pq.exch(1, n)
	pq.keys = pq.keys[0:n]
	delete(pq.positions, min)

	return min
}

func (pq *MinPQ) insert(node cipher.PubKey, dist int) {
	nodeDist := &NodeDist{node, dist}
	pq.keys = append(pq.keys, nodeDist)
	position := len(pq.keys) - 1
	pq.positions[node] = position
	pq.swim(position)
}

func (pq *MinPQ) decreaseKey(node cipher.PubKey, dist int) {
	position, found := pq.positions[node]
	if found {
		key := pq.keys[position]
		key.dist = dist
	}
}

func (pq *MinPQ) swim(k int) {
	for k > 1 && pq.less(k, k/2) {
		pq.exch(k, k/2)
		k /= 2
	}
}

func (pq *MinPQ) sink(k int) {
	n := len(pq.keys) - 1
	for 2*k < n {
		j := 2 * k
		if j < n && pq.less(j+1, j) {
			j++
		}
		if !pq.less(j, k) {
			break
		}
		pq.exch(k, j)
	}
}

func (pq *MinPQ) less(i, j int) bool {
	return pq.keys[i].dist < pq.keys[j].dist
}

func (pq *MinPQ) exch(i, j int) {
	pq.positions[pq.keys[i].node], pq.positions[pq.keys[j].node] = j, i
	pq.keys[i], pq.keys[j] = pq.keys[j], pq.keys[i]
}
