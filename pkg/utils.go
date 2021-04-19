package multiplex

func copyChan(in, out chan *Stream) {
	for s := range in {
		out <- s
	}
}
