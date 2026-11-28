// Пакет graphs — модель невзвешенного графа: вершины — целые ID 0..n-1,
// режимы directed и undirected. Два представления — список смежности и
// матрица смежности (matrix.go) — с преобразованиями в обе стороны.
package graphs

import "sort"

// Graph — невзвешенный граф на вершинах 0..n-1. Создавайте через New или
// NewFromEdges: списки соседей хранятся упорядоченными по возрастанию номеров
// вершин, а наружу отдаются копиями — извне граф изменить нельзя.
type Graph struct {
	n        int
	directed bool
	adj      [][]int
}

// Edge — ребро графа: пара вершин в диапазоне 0..n-1.
type Edge struct {
	U, V int
}

// New создаёт граф на n вершинах без рёбер. Пустой граф (n = 0) валиден;
// отрицательное n — ошибка ErrNegativeNumberOfVertices.
func New(n int, directed bool) (*Graph, error) {
	if n < 0 {
		return nil, ErrNegativeNumberOfVertices
	}
	return &Graph{
		n:        n,
		directed: directed,
		adj:      make([][]int, n),
	}, nil
}

// NewFromEdges строит граф из числа вершин и списка рёбер. Каждое ребро —
// пара вершин в диапазоне 0..n-1 (иначе ErrVertexNotFound); повторное ребро
// отклоняется ErrDuplicateEdge с учётом режима: в undirected (u, v) и (v, u) —
// одно и то же ребро, в directed противоположные направления — разные рёбра.
// Петля (u == v) разрешена.
func NewFromEdges(n int, edges []Edge, directed bool) (*Graph, error) {
	g, err := New(n, directed)
	if err != nil {
		return nil, err
	}
	for _, e := range edges {
		if err := g.AddEdge(e.U, e.V); err != nil {
			return nil, err
		}
	}
	return g, nil
}

// N возвращает число вершин графа (вершины 0..N-1).
func (g *Graph) N() int { return g.n }

// Directed сообщает режим графа: true — ориентированный, false —
// неориентированный.
func (g *Graph) Directed() bool { return g.directed }

// Neighbors возвращает копию списка соседей вершины v в порядке возрастания
// номеров вершин. Несуществующая вершина — ошибка ErrVertexNotFound, без
// паники. Изменение возвращённого слайса граф не меняет.
func (g *Graph) Neighbors(v int) ([]int, error) {
	if err := g.checkVertex(v); err != nil {
		return nil, err
	}
	nbrs := make([]int, len(g.adj[v]))
	copy(nbrs, g.adj[v])
	return nbrs, nil
}

// checkVertex проверяет, что v — существующая вершина графа.
func (g *Graph) checkVertex(v int) error {
	if v < 0 || v >= g.n {
		return ErrVertexNotFound
	}
	return nil
}

// AddEdge добавляет ребро u→v (в undirected — ребро {u, v}). Вершина вне
// диапазона 0..n-1 — ErrVertexNotFound; ребро, которое уже есть (с учётом
// режима), — ErrDuplicateEdge; петля разрешена. Соседи вершины u остаются
// в возрастающем порядке.
func (g *Graph) AddEdge(u, v int) error {
	if err := g.checkVertex(u); err != nil {
		return err
	}
	if err := g.checkVertex(v); err != nil {
		return err
	}
	if g.hasEdge(u, v) {
		return ErrDuplicateEdge
	}
	g.insertSorted(u, v)
	if !g.directed && u != v {
		g.insertSorted(v, u)
	}
	return nil
}

// RemoveEdge удаляет запись ребра u→v из списка соседей вершины u (в
// undirected — также запись v→u из списка вершины v). Отсутствующее ребро —
// no-op без ошибки; вершина вне диапазона — ErrVertexNotFound.
func (g *Graph) RemoveEdge(u, v int) error {
	if err := g.checkVertex(u); err != nil {
		return err
	}
	if err := g.checkVertex(v); err != nil {
		return err
	}
	g.adj[u] = removeFirstOccurrence(g.adj[u], v)
	if !g.directed && u != v {
		g.adj[v] = removeFirstOccurrence(g.adj[v], u)
	}
	return nil
}

// HasEdge сообщает, есть ли ребро u→v; для несуществующей вершины — false
// (предикат: без ошибки и без паники).
func (g *Graph) HasEdge(u, v int) bool {
	if err := g.checkVertex(u); err != nil {
		return false
	}
	if err := g.checkVertex(v); err != nil {
		return false
	}
	return g.hasEdge(u, v)
}

// Degree возвращает длину списка соседей вершины v (петля учитывается один
// раз — как один элемент списка); несуществующая вершина — ErrVertexNotFound.
func (g *Graph) Degree(v int) (int, error) {
	if err := g.checkVertex(v); err != nil {
		return 0, err
	}
	return len(g.adj[v]), nil
}

// hasEdge проверяет наличие ребра u→v бинарным поиском в упорядоченном
// списке соседей вершины u.
func (g *Graph) hasEdge(u, v int) bool {
	s := g.adj[u]
	i := sort.SearchInts(s, v)
	return i < len(s) && s[i] == v
}

// insertSorted вставляет v в список соседей вершины u с сохранением
// возрастающего порядка. Предусловие: ребра u→v ещё нет — публичные
// дубликаты отклоняет AddEdge проверкой hasEdge до вставки; встроенная
// проверка повтора ниже — страховка инварианта упорядоченности, а не
// разрешённый путь для дубликатов.
func (g *Graph) insertSorted(u, v int) {
	s := g.adj[u]
	i := sort.SearchInts(s, v)
	if i < len(s) && s[i] == v {
		return
	}
	s = append(s, 0)
	copy(s[i+1:], s[i:])
	s[i] = v
	g.adj[u] = s
}

// removeFirstOccurrence удаляет из слайса первую запись значения x и
// возвращает укороченный слайс; отсутствующее значение — тот же слайс.
func removeFirstOccurrence(s []int, x int) []int {
	for i, y := range s {
		if y == x {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}
