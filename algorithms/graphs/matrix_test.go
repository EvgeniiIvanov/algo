package graphs_test

import (
	"errors"
	"testing"

	"github.com/EvgeniiIvanov/algo/algorithms/graphs"
)

func equalMatrix(got, want [][]int) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if !equalInts(got[i], want[i]) {
			return false
		}
	}
	return true
}

func TestToAdjMatrixUndirected(t *testing.T) {
	// Несвязный граф с петлёй и изолированной вершиной: рёбра 0–1, 1–1, 3–4.
	g := mustNew(t, 5, false)
	mustAddEdge(t, g, 0, 1)
	mustAddEdge(t, g, 1, 1)
	mustAddEdge(t, g, 3, 4)

	want := [][]int{
		{0, 1, 0, 0, 0},
		{1, 1, 0, 0, 0},
		{0, 0, 0, 0, 0},
		{0, 0, 0, 0, 1},
		{0, 0, 0, 1, 0},
	}
	if got := g.ToAdjMatrix(); !equalMatrix(got, want) {
		t.Fatalf("ToAdjMatrix() = %v, ожидалось %v", got, want)
	}
}

func TestToAdjMatrixDirected(t *testing.T) {
	// В directed матрица несимметрична: ребро 0→1 не создаёт ячейку 1→0.
	g := mustNew(t, 3, true)
	mustAddEdge(t, g, 0, 1)
	mustAddEdge(t, g, 2, 1)

	want := [][]int{
		{0, 1, 0},
		{0, 0, 0},
		{0, 1, 0},
	}
	if got := g.ToAdjMatrix(); !equalMatrix(got, want) {
		t.Fatalf("ToAdjMatrix() = %v, ожидалось %v", got, want)
	}
}

func TestMatrixRoundTrip(t *testing.T) {
	for _, directed := range []bool{false, true} {
		// Исходный граф: треугольник, отдельное ребро, петля и изолированная
		// вершина 6. Список → матрица → список должен дать исходный граф.
		g := mustNew(t, 7, directed)
		mustAddEdge(t, g, 0, 1)
		mustAddEdge(t, g, 1, 2)
		mustAddEdge(t, g, 2, 0)
		mustAddEdge(t, g, 3, 4)
		mustAddEdge(t, g, 5, 5)

		m := g.ToAdjMatrix()
		round, err := graphs.FromAdjMatrix(m, directed)
		if err != nil {
			t.Fatalf("directed=%v: FromAdjMatrix неожиданная ошибка: %v", directed, err)
		}
		sameGraph(t, round, g)
	}
}

// sameGraph сравнивает графы по публичному API: число вершин, режим,
// все пары рёбер и списки соседей.
func sameGraph(t *testing.T, got, want *graphs.Graph) {
	t.Helper()
	if got.N() != want.N() {
		t.Fatalf("N() = %d, ожидалось %d", got.N(), want.N())
	}
	if got.Directed() != want.Directed() {
		t.Fatalf("Directed() = %v, ожидалось %v", got.Directed(), want.Directed())
	}
	for u := 0; u < want.N(); u++ {
		for v := 0; v < want.N(); v++ {
			if got.HasEdge(u, v) != want.HasEdge(u, v) {
				t.Fatalf("HasEdge(%d, %d) = %v, ожидалось %v", u, v, got.HasEdge(u, v), want.HasEdge(u, v))
			}
		}
	}
	for u := 0; u < want.N(); u++ {
		gotNbrs, wantNbrs := neighborsOf(t, got, u), neighborsOf(t, want, u)
		if !equalInts(gotNbrs, wantNbrs) {
			t.Fatalf("Neighbors(%d) = %v, ожидалось %v", u, gotNbrs, wantNbrs)
		}
	}
}

func TestNewFromEdgesDuplicateEdges(t *testing.T) {
	// В undirected (1, 0) — тот же дубликат, что повтор (0, 1).
	_, err := graphs.NewFromEdges(3, []graphs.Edge{{U: 0, V: 1}, {U: 1, V: 0}}, false)
	if !errors.Is(err, graphs.ErrDuplicateEdge) {
		t.Fatalf("NewFromEdges undirected [(0,1), (1,0)] = ошибка %v, ожидалась ErrDuplicateEdge", err)
	}
	_, err = graphs.NewFromEdges(3, []graphs.Edge{{U: 0, V: 1}, {U: 0, V: 1}}, false)
	if !errors.Is(err, graphs.ErrDuplicateEdge) {
		t.Fatalf("NewFromEdges undirected [(0,1), (0,1)] = ошибка %v, ожидалась ErrDuplicateEdge", err)
	}

	// В directed повтор того же направления — дубликат...
	_, err = graphs.NewFromEdges(3, []graphs.Edge{{U: 0, V: 1}, {U: 0, V: 1}}, true)
	if !errors.Is(err, graphs.ErrDuplicateEdge) {
		t.Fatalf("NewFromEdges directed [(0,1), (0,1)] = ошибка %v, ожидалась ErrDuplicateEdge", err)
	}
	// ...а противоположные направления — разные рёбра, оба принимаются.
	g, err := graphs.NewFromEdges(3, []graphs.Edge{{U: 0, V: 1}, {U: 1, V: 0}}, true)
	if err != nil {
		t.Fatalf("NewFromEdges directed [(0,1), (1,0)] неожиданная ошибка: %v", err)
	}
	if !g.HasEdge(0, 1) || !g.HasEdge(1, 0) {
		t.Fatalf("противоположные направления в directed не сохранились оба")
	}
}

func TestFromAdjMatrixAsymmetricUndirected(t *testing.T) {
	// Несимметричный вход: единица только в одной из пары симметричных
	// ячеек — в undirected это одно ребро (нормализация OR).
	m := [][]int{
		{0, 1, 0, 0}, // m[0][1] = 1, m[1][0] = 0
		{0, 0, 0, 0},
		{0, 0, 0, 0},
		{0, 0, 1, 0}, // m[3][2] = 1, m[2][3] = 0
	}
	g, err := graphs.FromAdjMatrix(m, false)
	if err != nil {
		t.Fatalf("FromAdjMatrix неожиданная ошибка: %v", err)
	}
	if !g.HasEdge(0, 1) || !g.HasEdge(1, 0) {
		t.Fatalf("ячейка m[0][1] не стала ребром {0, 1} в обе стороны")
	}
	if !g.HasEdge(2, 3) || !g.HasEdge(3, 2) {
		t.Fatalf("ячейка m[3][2] не стала ребром {2, 3} в обе стороны")
	}
	if got, want := neighborsOf(t, g, 0), []int{1}; !equalInts(got, want) {
		t.Fatalf("Neighbors(0) = %v, ожидалось %v", got, want)
	}
	if got, want := neighborsOf(t, g, 1), []int{0}; !equalInts(got, want) {
		t.Fatalf("Neighbors(1) = %v, ожидалось %v", got, want)
	}
	if got, want := neighborsOf(t, g, 2), []int{3}; !equalInts(got, want) {
		t.Fatalf("Neighbors(2) = %v, ожидалось %v", got, want)
	}
	if got, want := neighborsOf(t, g, 3), []int{2}; !equalInts(got, want) {
		t.Fatalf("Neighbors(3) = %v, ожидалось %v", got, want)
	}
}

func TestFromAdjMatrixNotSquare(t *testing.T) {
	// Рваная матрица: строка короче других — не квадрат V×V.
	_, err := graphs.FromAdjMatrix([][]int{{0, 1}, {1}}, false)
	if !errors.Is(err, graphs.ErrInvalidAdjacencyMatrix) {
		t.Fatalf("FromAdjMatrix(рваная матрица) = ошибка %v, ожидалась ErrInvalidAdjacencyMatrix", err)
	}
	// Число строк не равно длине строк.
	_, err = graphs.FromAdjMatrix([][]int{{0, 1, 0}, {1, 0, 0}}, true)
	if !errors.Is(err, graphs.ErrInvalidAdjacencyMatrix) {
		t.Fatalf("FromAdjMatrix(2×3) = ошибка %v, ожидалась ErrInvalidAdjacencyMatrix", err)
	}
}

func TestFromAdjMatrixEmpty(t *testing.T) {
	// Пустая (и nil) матрица — валидный пустой граф.
	for _, m := range [][][]int{nil, {}} {
		g, err := graphs.FromAdjMatrix(m, false)
		if err != nil {
			t.Fatalf("FromAdjMatrix(%v) неожиданная ошибка: %v", m, err)
		}
		if got := g.N(); got != 0 {
			t.Fatalf("N() = %d, ожидалось 0", got)
		}
	}
}

func TestMatrixMutationDoesNotAffectGraph(t *testing.T) {
	g := mustNew(t, 2, false)
	mustAddEdge(t, g, 0, 1)

	m := g.ToAdjMatrix()
	m[0][1] = 0
	m[1][1] = 1
	if !g.HasEdge(0, 1) || g.HasEdge(1, 1) {
		t.Fatalf("мутация возвращённой матрицы изменила граф")
	}
}
