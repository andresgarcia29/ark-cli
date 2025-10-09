package lib

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/andresgarcia29/ark-cli/logs"
)

// ParallelConfig controla los parámetros de paralelización
type ParallelConfig struct {
	// MaxWorkers define el número máximo de goroutines que pueden ejecutarse simultáneamente
	// Esto evita sobrecargar las APIs de AWS y el sistema local
	MaxWorkers int

	// Timeout define cuánto tiempo máximo puede tardar toda la operación paralela
	// Si se supera este tiempo, se cancelan todas las operaciones pendientes
	Timeout time.Duration

	// RateLimitDelay define el tiempo de espera entre el inicio de cada nueva tarea
	// Esto ayuda a evitar sobrecargar las APIs de AWS con demasiadas requests simultáneas
	RateLimitDelay time.Duration

	// MaxRetries define cuántas veces se reintentará una operación fallida
	// Útil para manejar errores temporales de red o límites de API
	MaxRetries int

	// RetryDelay define cuánto tiempo esperar entre reintentos
	RetryDelay time.Duration
}

// DefaultParallelConfig devuelve una configuración por defecto optimizada para AWS
func DefaultParallelConfig() ParallelConfig {
	return ParallelConfig{
		MaxWorkers:     10,                     // 10 workers concurrentes - balance entre velocidad y rate limits de AWS
		Timeout:        5 * time.Minute,        // 5 minutos máximo para operaciones paralelas
		RateLimitDelay: 100 * time.Millisecond, // 100ms entre tareas para respetar rate limits
		MaxRetries:     3,                      // 3 reintentos para operaciones fallidas
		RetryDelay:     1 * time.Second,        // 1 segundo entre reintentos
	}
}

// ConservativeConfig devuelve una configuración más conservadora para ambientes sensibles
func ConservativeConfig() ParallelConfig {
	return ParallelConfig{
		MaxWorkers:     5,                      // Menos workers para ser más conservador
		Timeout:        10 * time.Minute,       // Más tiempo para operaciones
		RateLimitDelay: 500 * time.Millisecond, // Más delay entre requests
		MaxRetries:     5,                      // Más reintentos
		RetryDelay:     2 * time.Second,        // Más tiempo entre reintentos
	}
}

// AggressiveConfig devuelve una configuración más agresiva para máximo rendimiento
func AggressiveConfig() ParallelConfig {
	return ParallelConfig{
		MaxWorkers:     20,                     // Más workers para máximo paralelismo
		Timeout:        3 * time.Minute,        // Menos tiempo para operaciones
		RateLimitDelay: 50 * time.Millisecond,  // Menos delay entre requests
		MaxRetries:     2,                      // Menos reintentos
		RetryDelay:     500 * time.Millisecond, // Menos tiempo entre reintentos
	}
}

// WorkerPool representa un pool de workers para ejecutar tareas en paralelo
type WorkerPool struct {
	// maxWorkers controla cuántas goroutines pueden ejecutarse simultáneamente
	maxWorkers int
	// semaphore es un channel que actúa como semáforo para controlar la concurrencia
	// Cuando está lleno, las nuevas goroutines esperan hasta que se libere espacio
	semaphore chan struct{}
}

// NewWorkerPool crea un nuevo pool de workers con el número máximo especificado
func NewWorkerPool(maxWorkers int) *WorkerPool {
	return &WorkerPool{
		maxWorkers: maxWorkers,
		// Creamos un channel con capacidad igual al número máximo de workers
		// Esto actúa como un semáforo: cuando está lleno, las nuevas tareas esperan
		semaphore: make(chan struct{}, maxWorkers),
	}
}

// Execute ejecuta una función en el pool de workers
// Esta función bloquea hasta que hay un worker disponible
func (wp *WorkerPool) Execute(ctx context.Context, fn func() error) error {
	select {
	// Intentamos adquirir un slot en el semáforo
	case wp.semaphore <- struct{}{}:
		// ¡Tenemos un slot! Ejecutamos la función
		defer func() {
			// Al terminar, liberamos el slot para que otro worker pueda usarlo
			<-wp.semaphore
		}()
		return fn()

	// Si el contexto se cancela mientras esperamos un slot, retornamos error
	case <-ctx.Done():
		return ctx.Err()
	}
}

// AccountResult representa el resultado de procesar una cuenta específica
type AccountResult struct {
	// AccountID identifica qué cuenta se procesó
	AccountID string
	// Data contiene los datos obtenidos (puede ser []EKSCluster, []Role, etc.)
	Data interface{}
	// Error contiene cualquier error que ocurrió durante el procesamiento
	Error error
}

// GetWorkerPool es un alias para NewWorkerPool para facilitar el uso externo
func GetWorkerPool(maxWorkers int) *WorkerPool {
	return NewWorkerPool(maxWorkers)
}

// ExecuteWithRetry ejecuta una función con reintentos automáticos
// Esta función es útil para operaciones que pueden fallar temporalmente (red, rate limits, etc.)
func ExecuteWithRetry(ctx context.Context, config ParallelConfig, operation func() error) error {
	logger := logs.GetLogger()
	var lastErr error

	// Intentamos la operación hasta MaxRetries + 1 veces (intento inicial + reintentos)
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Si no es el primer intento, esperamos antes de reintentar
		if attempt > 0 {
			logger.Debugw("Reintentando operación",
				"attempt", attempt,
				"max_retries", config.MaxRetries,
				"delay", config.RetryDelay)

			// Usamos select para respetar el contexto durante la espera
			select {
			case <-time.After(config.RetryDelay):
				// Tiempo de espera completado, continuamos
			case <-ctx.Done():
				// El contexto fue cancelado, retornamos el error
				return fmt.Errorf("operación cancelada durante reintento: %w", ctx.Err())
			}
		}

		// Ejecutamos la operación
		err := operation()
		if err == nil {
			// ¡Éxito! No necesitamos más reintentos
			if attempt > 0 {
				logger.Infow("Operación exitosa después de reintentos",
					"successful_attempt", attempt+1)
			}
			return nil
		}

		// Guardamos el error para reportarlo si todos los intentos fallan
		lastErr = err

		// Si es el último intento, no mostramos mensaje de reintento
		if attempt < config.MaxRetries {
			logger.Warnw("Intento falló, reintentando",
				"attempt", attempt+1,
				"error", err)
		}
	}

	// Todos los intentos fallaron
	logger.Errorw("Operación falló después de todos los reintentos",
		"attempts", config.MaxRetries+1,
		"error", lastErr)
	return fmt.Errorf("operación falló después de %d intentos: %w", config.MaxRetries+1, lastErr)
}

// RateLimiter controla la velocidad de ejecución de operaciones
type RateLimiter struct {
	// delay es el tiempo de espera entre operaciones
	delay time.Duration
	// lastExecution guarda cuándo se ejecutó la última operación
	lastExecution time.Time
	// mutex protege el acceso concurrente a lastExecution
	mutex sync.Mutex
}

// NewRateLimiter crea un nuevo rate limiter con el delay especificado
func NewRateLimiter(delay time.Duration) *RateLimiter {
	return &RateLimiter{
		delay: delay,
	}
}

// Wait espera el tiempo necesario para respetar el rate limit
// Esta función asegura que no ejecutemos operaciones demasiado rápido
func (rl *RateLimiter) Wait(ctx context.Context) error {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	// Calculamos cuánto tiempo necesitamos esperar
	now := time.Now()
	timeSinceLastExecution := now.Sub(rl.lastExecution)

	if timeSinceLastExecution < rl.delay {
		// Necesitamos esperar más tiempo
		waitTime := rl.delay - timeSinceLastExecution

		// Liberamos el mutex durante la espera para no bloquear otros workers
		rl.mutex.Unlock()

		select {
		case <-time.After(waitTime):
			// Tiempo de espera completado
		case <-ctx.Done():
			// El contexto fue cancelado
			rl.mutex.Lock() // Re-adquirimos el mutex para el defer
			return ctx.Err()
		}

		// Re-adquirimos el mutex
		rl.mutex.Lock()
	}

	// Actualizamos el tiempo de última ejecución
	rl.lastExecution = time.Now()
	return nil
}

// ProcessAccountsInParallel procesa múltiples cuentas AWS en paralelo
// Esta función es genérica y puede usarse para cualquier operación que necesite
// ejecutarse en paralelo para múltiples cuentas
func ProcessAccountsInParallel[T any](
	ctx context.Context,
	accounts []string,
	config ParallelConfig,
	processor func(ctx context.Context, accountID string) (T, error),
) (map[string]T, []error) {

	// Creamos un contexto con timeout para toda la operación
	// Si la operación tarda más del timeout configurado, se cancelará automáticamente
	timeoutCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel() // Importante: siempre cancelar el contexto al finalizar

	// WaitGroup nos permite esperar a que todas las goroutines terminen
	var wg sync.WaitGroup

	// Channel para recibir los resultados de cada goroutine
	// Tiene capacidad igual al número de cuentas para evitar bloqueos
	resultChan := make(chan AccountResult, len(accounts))

	// Creamos el pool de workers para controlar la concurrencia
	workerPool := NewWorkerPool(config.MaxWorkers)

	// Creamos un rate limiter para controlar la velocidad de las requests
	rateLimiter := NewRateLimiter(config.RateLimitDelay)

	logger := logs.GetLogger()
	logger.Infow("Iniciando procesamiento paralelo",
		"total_accounts", len(accounts),
		"max_workers", config.MaxWorkers,
		"rate_limit", config.RateLimitDelay,
		"timeout", config.Timeout)

	// Lanzamos una goroutine para cada cuenta
	for _, accountID := range accounts {
		// Incrementamos el contador del WaitGroup antes de lanzar la goroutine
		wg.Add(1)

		// Capturamos el valor de accountID en una variable local
		// Esto es importante en Go para evitar problemas con closures
		currentAccountID := accountID

		// Lanzamos la goroutine
		go func() {
			// Decrementamos el contador del WaitGroup cuando terminemos
			defer wg.Done()

			logger.Debugf("Procesando cuenta: %s", currentAccountID)

			// Ejecutamos el procesamiento en el worker pool
			// Esto controlará la concurrencia automáticamente
			err := workerPool.Execute(timeoutCtx, func() error {
				// Primero esperamos para respetar el rate limit
				// Esto previene sobrecargar las APIs de AWS
				if err := rateLimiter.Wait(timeoutCtx); err != nil {
					return fmt.Errorf("rate limit cancelado: %w", err)
				}

				// Ahora ejecutamos la operación con reintentos automáticos
				var result T
				var processingErr error

				retryErr := ExecuteWithRetry(timeoutCtx, config, func() error {
					// Aquí ejecutamos la función de procesamiento específica
					var err error
					result, err = processor(timeoutCtx, currentAccountID)
					processingErr = err
					return err
				})

				// Si los reintentos fallaron, usamos el último error
				if retryErr != nil {
					processingErr = retryErr
				}

				// Enviamos el resultado al channel
				// Usamos select para manejar el caso donde el contexto se cancela
				select {
				case resultChan <- AccountResult{
					AccountID: currentAccountID,
					Data:      result,
					Error:     processingErr,
				}:
					// Resultado enviado exitosamente
					if processingErr != nil {
						logger.Errorw("Error procesando cuenta",
							"account_id", currentAccountID,
							"error", processingErr)
					} else {
						logger.Infow("Cuenta procesada exitosamente",
							"account_id", currentAccountID)
					}
				case <-timeoutCtx.Done():
					// El contexto fue cancelado, no podemos enviar el resultado
					return timeoutCtx.Err()
				}
				return nil
			})

			// Si hubo error en el worker pool (por timeout), enviamos el error
			if err != nil {
				select {
				case resultChan <- AccountResult{
					AccountID: currentAccountID,
					Data:      *new(T), // valor cero del tipo T
					Error:     err,
				}:
				case <-timeoutCtx.Done():
					// No podemos enviar, pero no importa porque ya estamos cancelando
				}
			}
		}()
	}

	// Lanzamos una goroutine para cerrar el channel cuando todas las tareas terminen
	go func() {
		// Esperamos a que todas las goroutines terminen
		wg.Wait()
		// Cerramos el channel para indicar que no habrá más resultados
		close(resultChan)
		logger.Debug("Todas las cuentas han sido procesadas")
	}()

	// Recolectamos todos los resultados del channel
	results := make(map[string]T)
	var errors []error

	// Leemos del channel hasta que se cierre
	for result := range resultChan {
		if result.Error != nil {
			// Si hubo error, lo agregamos a la lista de errores
			errors = append(errors, fmt.Errorf("cuenta %s: %w", result.AccountID, result.Error))
		} else {
			// Si fue exitoso, agregamos el resultado al mapa
			results[result.AccountID] = result.Data.(T)
		}
	}

	logger.Infow("Procesamiento paralelo completado",
		"successful", len(results),
		"errors", len(errors))

	return results, errors
}
