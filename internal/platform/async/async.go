// Package async provee un pool de workers in-process para ejecutar tareas en
// segundo plano sin bloquear el request principal (RNF-02 / D-06).
//
// Uso típico desde un handler:
//
//	jobs.Enqueue(async.Job{
//	    Name: "notificar-transportistas",
//	    Run:  func(ctx context.Context) error { return svc.NotificarCompatibles(ctx, solicitudID) },
//	})
//
// Los jobs deben ser IDEMPOTENTES: ante fallo se logea y (opcionalmente) se
// reintenta; no hay garantía de exactly-once.
package async

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Job es una unidad de trabajo en segundo plano.
type Job struct {
	Name       string
	Run        func(ctx context.Context) error
	MaxRetries int           // 0 = sin reintentos
	Backoff    time.Duration // espera entre reintentos (default 2s)
}

// Enqueuer es la interfaz mínima que consumen los servicios (facilita el testing).
type Enqueuer interface {
	Enqueue(job Job) bool
}

// Pool ejecuta jobs con un número fijo de workers.
type Pool struct {
	queue   chan Job
	workers int
	log     *slog.Logger
	wg      sync.WaitGroup
	baseCtx context.Context
	cancel  context.CancelFunc
}

func NewPool(workers, queueSize int, log *slog.Logger) *Pool {
	if workers < 1 {
		workers = 1
	}
	if queueSize < 1 {
		queueSize = 1
	}
	return &Pool{
		queue:   make(chan Job, queueSize),
		workers: workers,
		log:     log,
	}
}

// Start arranca los workers. El ctx recibido acota la vida del pool.
func (p *Pool) Start(ctx context.Context) {
	p.baseCtx, p.cancel = context.WithCancel(ctx)
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	p.log.Info("pool async iniciado", "workers", p.workers, "queue", cap(p.queue))
}

// Enqueue encola un job. Devuelve false si la cola está llena (el caller decide si
// ejecutar sincrónico como fallback o descartar).
func (p *Pool) Enqueue(job Job) bool {
	select {
	case p.queue <- job:
		return true
	default:
		p.log.Warn("cola async llena, job descartado", "job", job.Name)
		return false
	}
}

// Stop cierra la cola y espera a que terminen los jobs en vuelo.
func (p *Pool) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	close(p.queue)
	p.wg.Wait()
	p.log.Info("pool async detenido")
}

func (p *Pool) worker(id int) {
	defer p.wg.Done()
	for job := range p.queue {
		p.execute(id, job)
	}
}

func (p *Pool) execute(workerID int, job Job) {
	backoff := job.Backoff
	if backoff == 0 {
		backoff = 2 * time.Second
	}
	for attempt := 0; ; attempt++ {
		start := time.Now()
		err := safeRun(p.baseCtx, job)
		if err == nil {
			p.log.Debug("job ok", "job", job.Name, "worker", workerID, "dur", time.Since(start))
			return
		}
		if attempt >= job.MaxRetries {
			p.log.Error("job falló definitivamente", "job", job.Name, "intentos", attempt+1, "err", err)
			return
		}
		p.log.Warn("job falló, reintentando", "job", job.Name, "intento", attempt+1, "err", err)
		select {
		case <-time.After(backoff):
		case <-p.baseCtx.Done():
			return
		}
	}
}

func safeRun(ctx context.Context, job Job) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = &panicError{v: r}
		}
	}()
	return job.Run(ctx)
}

type panicError struct{ v any }

func (e *panicError) Error() string { return "panic en job async" }
