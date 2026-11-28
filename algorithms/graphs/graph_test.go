package graphs_test

import (
	"errors"
	"testing"

	"github.com/EvgeniiIvanov/algo/algorithms/graphs"
)

func mustNew(t *testing.T, n int, directed bool) *graphs.Graph {
	t.Helper()
	g, err := graphs.New(n, directed)
	if err != nil {
		t.Fatalf("New(%d, %v) неожиданная ошибка: %v", n, directed, err)
	}
	return g
}

func mustAddEdge(t *testing.T, g *graphs.Graph, u, v int) {
	t.Helper()
	if err := g.AddEdge(u, v); err != nil {
		t.Fatalf("AddEdge(%d, %d) неожиданная ошибка: %v", u, v, err)
	}
}

func neighborsOf(t *testing.T, g *graphs.Graph, v int) []int {
	t.Helper()
	nbrs, err := g.Neighbors(v)
	if err != nil {
		t.Fatalf("Neighbors(%d) неожиданная ошибка: %v", v, err)
	}
	return nbrs
}

func equalInts(got, want []int) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func TestNewEmptyGraph(t *testing.T) {
	for _, directed := range []bool{false, true} {
		g := mustNew(t, 0, directed)
		if got := g.N(); got != 0 {
			t.Fatalf("N() = %d, ожидалось 0", got)
		}
		if got := g.Directed(); got != directed {
			t.Fatalf("Directed() = %v, ожидалось %v", got, directed)
		}
		// В графе нет ни одной вершины — любой доступ к соседям даёт ошибку.
		if _, err := g.Neighbors(0); !errors.Is(err, graphs.ErrVertexNotFound) {
			t.Fatalf("Neighbors(0) в пустом графе = ошибка %v, ожидалась ErrVertexNotFound", err)
		}
	}
}

func TestNewNegativeVertices(t *testing.T) {
	// Отрицательное число вершин — ошибка; ноль валиден (пустой граф).
	if _, err := graphs.New(-1, false); !errors.Is(err, graphs.ErrNegativeNumberOfVertices) {
		t.Fatalf("New(-1) = ошибка %v, ожидалась ErrNegativeNumberOfVertices", err)
	}
	if _, err := graphs.NewFromEdges(-1, nil, true); !errors.Is(err, graphs.ErrNegativeNumberOfVertices) {
		t.Fatalf("NewFromEdges(-1, nil) = ошибка %v, ожидалась ErrNegativeNumberOfVertices", err)
	}
}

func TestSingleVertex(t *testing.T) {
	g := mustNew(t, 1, false)
	if got := g.N(); got != 1 {
		t.Fatalf("N() = %d, ожидалось 1", got)
	}
	if got := neighborsOf(t, g, 0); len(got) != 0 {
		t.Fatalf("Neighbors(0) = %v, ожидался пустой список", got)
	}
	m := g.ToAdjMatrix()
	if len(m) != 1 || len(m[0]) != 1 || m[0][0] != 0 {
		t.Fatalf("ToAdjMatrix() = %v, ожидалось [[0]]", m)
	}
}

func TestIsolatedVerticesAndDisconnected(t *testing.T) {
	// Несвязный граф: компоненты {0, 1} и {3, 4}, вершина 2 изолирована.
	g := mustNew(t, 5, false)
	mustAddEdge(t, g, 0, 1)
	mustAddEdge(t, g, 3, 4)

	if got := g.N(); got != 5 {
		t.Fatalf("N() = %d, ожидалось 5", got)
	}
	if got := neighborsOf(t, g, 2); len(got) != 0 {
		t.Fatalf("Neighbors(2) = %v, ожидался пустой список (вершина изолирована)", got)
	}
	if got, want := neighborsOf(t, g, 0), []int{1}; !equalInts(got, want) {
		t.Fatalf("Neighbors(0) = %v, ожидалось %v", got, want)
	}
	if got, want := neighborsOf(t, g, 1), []int{0}; !equalInts(got, want) {
		t.Fatalf("Neighbors(1) = %v, ожидалось %v", got, want)
	}
	if got, want := neighborsOf(t, g, 3), []int{4}; !equalInts(got, want) {
		t.Fatalf("Neighbors(3) = %v, ожидалось %v", got, want)
	}
	if got, want := neighborsOf(t, g, 4), []int{3}; !equalInts(got, want) {
		t.Fatalf("Neighbors(4) = %v, ожидалось %v", got, want)
	}
}

func TestUndirectedDuplicateEdge(t *testing.T) {
	g := mustNew(t, 3, false)
	mustAddEdge(t, g, 1, 2)

	// В undirected (2, 1) — то же ребро, что (1, 2): дубликат отклоняется ошибкой.
	if err := g.AddEdge(2, 1); !errors.Is(err, graphs.ErrDuplicateEdge) {
		t.Fatalf("AddEdge(2, 1) после AddEdge(1, 2) = ошибка %v, ожидалась ErrDuplicateEdge", err)
	}
	// Граф не изменился: ребро по-прежнему одно, ни лишних соседей.
	if got, want := neighborsOf(t, g, 1), []int{2}; !equalInts(got, want) {
		t.Fatalf("после отклонённого дубликата Neighbors(1) = %v, ожидалось %v", got, want)
	}
	if got, want := neighborsOf(t, g, 2), []int{1}; !equalInts(got, want) {
		t.Fatalf("после отклонённого дубликата Neighbors(2) = %v, ожидалось %v", got, want)
	}
}

func TestDirectedOppositeEdges(t *testing.T) {
	g := mustNew(t, 3, true)
	mustAddEdge(t, g, 1, 2)

	// В directed противоположные направления — разные рёбра: оба принимаются.
	mustAddEdge(t, g, 2, 1)
	if got, want := neighborsOf(t, g, 1), []int{2}; !equalInts(got, want) {
		t.Fatalf("Neighbors(1) = %v, ожидалось %v", got, want)
	}
	if got, want := neighborsOf(t, g, 2), []int{1}; !equalInts(got, want) {
		t.Fatalf("Neighbors(2) = %v, ожидалось %v", got, want)
	}

	// Повтор того же направления — дубликат в обе стороны.
	if err := g.AddEdge(1, 2); !errors.Is(err, graphs.ErrDuplicateEdge) {
		t.Fatalf("повторный AddEdge(1, 2) = ошибка %v, ожидалась ErrDuplicateEdge", err)
	}
	if err := g.AddEdge(2, 1); !errors.Is(err, graphs.ErrDuplicateEdge) {
		t.Fatalf("повторный AddEdge(2, 1) = ошибка %v, ожидалась ErrDuplicateEdge", err)
	}
}

func TestEdgeOutOfRange(t *testing.T) {
	g := mustNew(t, 3, false)
	if err := g.AddEdge(0, 3); !errors.Is(err, graphs.ErrVertexNotFound) {
		t.Fatalf("AddEdge(0, 3) = ошибка %v, ожидалась ErrVertexNotFound", err)
	}
	if err := g.AddEdge(-1, 0); !errors.Is(err, graphs.ErrVertexNotFound) {
		t.Fatalf("AddEdge(-1, 0) = ошибка %v, ожидалась ErrVertexNotFound", err)
	}
	// Ребро вне диапазона отклоняется и на входе NewFromEdges.
	_, err := graphs.NewFromEdges(3, []graphs.Edge{{U: 0, V: 1}, {U: 1, V: 5}}, false)
	if !errors.Is(err, graphs.ErrVertexNotFound) {
		t.Fatalf("NewFromEdges с ребром вне диапазона = ошибка %v, ожидалась ErrVertexNotFound", err)
	}
}

func TestSelfLoop(t *testing.T) {
	for _, directed := range []bool{false, true} {
		g := mustNew(t, 3, directed)
		mustAddEdge(t, g, 2, 2)

		if !g.HasEdge(2, 2) {
			t.Fatalf("directed=%v: петля не сохранилась", directed)
		}
		deg, err := g.Degree(2)
		if err != nil || deg != 1 {
			t.Fatalf("directed=%v: Degree(2) = (%d, %v), ожидалось (1, nil)", directed, deg, err)
		}
		m := g.ToAdjMatrix()
		if m[2][2] != 1 {
			t.Fatalf("directed=%v: m[2][2] = %d, ожидалось 1", directed, m[2][2])
		}
	}
}

func TestNeighborsAscendingOrder(t *testing.T) {
	// Рёбра добавляются не по порядку — соседи всё равно перечисляются
	// по возрастанию номеров вершин.
	g := mustNew(t, 5, false)
	for _, e := range []graphs.Edge{{U: 3, V: 4}, {U: 3, V: 1}, {U: 3, V: 0}, {U: 3, V: 2}} {
		mustAddEdge(t, g, e.U, e.V)
	}
	if got, want := neighborsOf(t, g, 3), []int{0, 1, 2, 4}; !equalInts(got, want) {
		t.Fatalf("Neighbors(3) = %v, ожидалось %v (по возрастанию)", got, want)
	}

	dg := mustNew(t, 4, true)
	for _, e := range []graphs.Edge{{U: 0, V: 3}, {U: 0, V: 1}, {U: 0, V: 2}} {
		mustAddEdge(t, dg, e.U, e.V)
	}
	if got, want := neighborsOf(t, dg, 0), []int{1, 2, 3}; !equalInts(got, want) {
		t.Fatalf("Neighbors(0) = %v, ожидалось %v (по возрастанию)", got, want)
	}
}

func TestNeighborsInvalidVertex(t *testing.T) {
	g := mustNew(t, 2, false)
	for _, v := range []int{-1, 2} {
		if _, err := g.Neighbors(v); !errors.Is(err, graphs.ErrVertexNotFound) {
			t.Fatalf("Neighbors(%d) = ошибка %v, ожидалась ErrVertexNotFound (без паники)", v, err)
		}
	}
}

func TestNeighborsReturnsCopy(t *testing.T) {
	g := mustNew(t, 3, false)
	mustAddEdge(t, g, 0, 1)
	mustAddEdge(t, g, 0, 2)

	nbrs := neighborsOf(t, g, 0)
	nbrs[0] = 100 // мутация копии не должна затрагивать граф
	if got, want := neighborsOf(t, g, 0), []int{1, 2}; !equalInts(got, want) {
		t.Fatalf("после мутации копии Neighbors(0) = %v, ожидалось %v", got, want)
	}
}

func TestRemoveEdge(t *testing.T) {
	g := mustNew(t, 3, false)
	mustAddEdge(t, g, 0, 1)
	mustAddEdge(t, g, 1, 1)

	if err := g.RemoveEdge(0, 1); err != nil {
		t.Fatalf("RemoveEdge(0, 1) неожиданная ошибка: %v", err)
	}
	if g.HasEdge(0, 1) || g.HasEdge(1, 0) {
		t.Fatalf("неориентированное ребро удалилось не в обе стороны")
	}
	if err := g.RemoveEdge(1, 1); err != nil {
		t.Fatalf("RemoveEdge(1, 1) неожиданная ошибка: %v", err)
	}
	if g.HasEdge(1, 1) {
		t.Fatalf("петля не удалилась")
	}
	// Удаление отсутствующего ребра — не ошибка, граф не меняется.
	if err := g.RemoveEdge(2, 2); err != nil {
		t.Fatalf("RemoveEdge(2, 2) без ребра = ошибка %v, ожидался no-op", err)
	}
	if got := neighborsOf(t, g, 2); len(got) != 0 {
		t.Fatalf("после no-op удаления Neighbors(2) = %v, ожидался пустой список", got)
	}
}
