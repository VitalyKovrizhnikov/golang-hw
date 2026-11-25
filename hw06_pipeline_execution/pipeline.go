package hw06pipelineexecution

type (
	In  = <-chan interface{}
	Out = In
	Bi  = chan interface{}
)

type Stage func(in In) (out Out)

func ExecutePipeline(in In, done In, stages ...Stage) Out {
	if done == nil {
		out := in
		for _, stage := range stages {
			out = stage(out)
		}

		return out
	}

	out := relay(in, done)
	for _, stage := range stages {
		out = relay(stage(out), done)
	}

	return out
}

func relay(in In, done In) Out {
	out := make(Bi)

	go func() {
		defer close(out)

		for {
			if isClosed(done) {
				go drain(in)
				return
			}

			select {
			case value, ok := <-in:
				if !ok {
					return
				}

				select {
				case out <- value:
				case <-done:
					go drain(in)
					return
				}
			case <-done:
				go drain(in)
				return
			}
		}
	}()

	return out
}

func isClosed(done In) bool {
	select {
	case <-done:
		return true
	default:
		return false
	}
}

func drain(in In) {
	for range in {
		// Discard values so upstream stages can finish and release their goroutines.
	}
}
