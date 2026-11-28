package graphs

// ToAdjMatrix возвращает матрицу смежности V×V: m[u][v] = 1, если ребро u→v
// есть, иначе 0. В directed матрица в общем случае несимметрична, в undirected
// — симметрична; у петли единица на диагонали. Матрица — независимая копия:
// её изменение граф не меняет.
func (g *Graph) ToAdjMatrix() [][]int {
	m := make([][]int, g.n)
	for i := range m {
		m[i] = make([]int, g.n)
	}
	for u := 0; u < g.n; u++ {
		for _, v := range g.adj[u] {
			m[u][v] = 1
			if !g.directed && u != v {
				m[v][u] = 1
			}
		}
	}
	return m
}

// FromAdjMatrix строит граф из квадратной матрицы смежности V×V: ненулевая
// ячейка — ребро. Матрица обязана быть квадратной (число строк равно длине
// каждой строки) — иначе ErrInvalidAdjacencyMatrix. Режим передаёт вызывающий
// (directed — параметр): сама матрица направление не кодирует. В directed
// ячейки читаются как есть; в undirected несимметричный вход нормализуется —
// единица в любой из пары симметричных ячеек m[u][v] / m[v][u] даёт одно
// ребро {u, v}. Петли и изолированные вершины сохраняются.
func FromAdjMatrix(m [][]int, directed bool) (*Graph, error) {
	n := len(m)
	for _, row := range m {
		if len(row) != n {
			return nil, ErrInvalidAdjacencyMatrix
		}
	}
	g, err := New(n, directed)
	if err != nil {
		return nil, err
	}
	for u := 0; u < n; u++ {
		for v := 0; v < n; v++ {
			has := m[u][v] != 0
			if !directed && !has {
				has = m[v][u] != 0
			}
			if !has {
				continue
			}
			if u == v {
				err = g.AddEdge(u, u)
			} else if directed {
				err = g.AddEdge(u, v)
			} else if u < v {
				err = g.AddEdge(u, v)
			} else {
				continue
			}
			if err != nil {
				return nil, err
			}
		}
	}
	return g, nil
}
