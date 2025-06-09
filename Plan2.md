### Подробный план реализации онлайн-редактора кода на Go

---
### Пошаговая последовательность работ:
1. День 1-2: Настройка Docker, БД, инициализация проекта
2. День 3-5: Реализация выполнения кода + тесты безопасности
3. День 6-7: HTTP API + интеграция с фронтендом
4. День 8: Система сохранения сниппетов
5. День 9: Оптимизация (кеш, пул контейнеров)
6. День 10: Нагрузочное тестирование
7. День 11: Документирование API (Swagger)
8. День 12: Подготовка к деплою

---

#### Этап 1: Подготовка инфраструктуры (1-2 дня)
1. **Установка зависимостей**:
   - Docker Engine + настройка пользовательских прав
   - PostgreSQL и Redis
   - Go 1.20+ (с поддержкой модулей)

2. **Инициализация проекта**:
   ```bash
   mkdir online-editor && cd online-editor
   go mod init github.com/yourname/online-editor
   ```

3. **Структура проекта**:
   ```
   /cmd
     /server        # основной сервер
   /internal
     /app           # бизнес-логика
       /executor    # выполнение кода
       /snippet     # управление сниппетами
     /transport     # обработчики HTTP
       /rest
     /config        # конфигурация
     /storage       # взаимодействие с БД
   /pkg
     /docker        # обёртка Docker API
     /monaco        # интеграция с редактором
   /deploy          # docker-compose, k8s манифесты
   ```

---

#### Этап 2: Реализация выполнения кода (3-4 дня)

1. **Сервис Docker Executor**:
   ```go
   // internal/app/executor/service.go
   type CodeExecutor struct {
     dockerClient *docker.Client
     config       Config
   }

   func (e *CodeExecutor) RunGoCode(code string) (string, error) {
     ctx, cancel := context.WithTimeout(context.Background(), e.config.Timeout)
     defer cancel()
     
     // Создание временного файла
     tmpFile, err := e.createTempFile(code)
     if err != nil { /* обработка */ }
     
     // Конфигурация контейнера
     containerConfig := e.createContainerConfig(tmpFile)
     
     // Запуск контейнера
     resp, err := e.dockerClient.ContainerCreate(ctx, &containerConfig, ...)
     if err != nil { /* обработка */ }
     
     // Получение логов
     logs, err := e.getContainerLogs(ctx, resp.ID)
     return logs, nil
   }
   ```

2. **Безопасное выполнение**:
   - Ограничение ресурсов:
     ```go
     hostConfig := &container.HostConfig{
       Memory:     100 * 1024 * 1024,  // 100MB
       CPUQuota:   50000,              // 50% CPU
       PidsLimit:  100,
       AutoRemove: true,
       SecurityOpt: []string{"no-new-privileges"},
     }
     ```
   - Запрет сетевого доступа:
     ```go
     networkConfig := &network.NetworkingConfig{
       EndpointsConfig: map[string]*network.EndpointSettings{
         "none": {},
       },
     }
     ```

---

#### Этап 3: Бэкенд-сервер (2-3 дня)

1. **Конфигурация**:
   ```go
   // internal/config/config.go
   type Config struct {
     HTTPPort       string        `env:"HTTP_PORT,default=8080"`
     DockerAPIVer   string        `env:"DOCKER_API_VERSION,default=1.41"`
     ExecTimeout    time.Duration `env:"EXEC_TIMEOUT,default=10s"`
     MaxConcurrent  int           `env:"MAX_CONCURRENT,default=5"`
   }
   ```

2. **HTTP-обработчики**:
   ```go
   // internal/transport/rest/handlers.go
   func RunHandler(exec *executor.CodeExecutor) gin.HandlerFunc {
     return func(c *gin.Context) {
       var request struct {
         Code string `json:"code" binding:"required,min=10"`
         Lang string `json:"lang" binding:"required,oneof=go python"`
       }
       
       if err := c.BindJSON(&request); err != nil {
         c.JSON(400, gin.H{"error": "Invalid request"})
         return
       }
       
       output, err := exec.Run(request.Code, request.Lang)
       if err != nil {
         c.JSON(500, gin.H{"error": err.Error()})
         return
       }
       
       c.JSON(200, gin.H{"output": output})
     }
   }
   ```

3. **Rate Limiting**:
   ```go
   // middleware/limiter.go
   func RateLimiter(maxRequests int) gin.HandlerFunc {
     limiter := rate.NewLimiter(rate.Every(time.Minute), maxRequests)
     return func(c *gin.Context) {
       if !limiter.Allow() {
         c.AbortWithStatusJSON(429, gin.H{"error": "Too many requests"})
         return
       }
       c.Next()
     }
   }
   ```

---

#### Этап 4: Фронтенд-интеграция (2 дня)

1. **React + Monaco Editor**:
   ```jsx
   // frontend/src/Editor.js
   import Editor from "@monaco-editor/react";
   
   function CodeEditor({ onRun }) {
     const [code, setCode] = useState("package main\n\nfunc main() {\n\tprintln(\"Hello\")\n}");
     
     return (
       <div>
         <Editor
           height="60vh"
           language="go"
           value={code}
           onChange={setCode}
         />
         <button onClick={() => onRun(code)}>Run</button>
       </div>
     );
   }
   ```

2. **Взаимодействие с API**:
   ```js
   // frontend/src/api.js
   export const runCode = async (code, lang) => {
     const response = await fetch('/api/run', {
       method: 'POST',
       headers: {'Content-Type': 'application/json'},
       body: JSON.stringify({ code, lang })
     });
     return response.json();
   };
   ```

---

#### Этап 5: Управление сниппетами (1-2 дня)

1. **PostgreSQL-модель**:
   ```sql
   CREATE TABLE snippets (
     id UUID PRIMARY KEY,
     code TEXT NOT NULL,
     language VARCHAR(10) NOT NULL,
     created_at TIMESTAMP DEFAULT NOW(),
     expires_at TIMESTAMP
   );
   ```

2. **Хранилище**:
   ```go
   // internal/storage/postgres/snippet.go
   func (s *SnippetStorage) Save(ctx context.Context, code string, lang string) (string, error) {
     id := uuid.New()
     _, err := s.db.ExecContext(ctx,
       `INSERT INTO snippets (id, code, language) VALUES ($1, $2, $3)`,
       id, code, lang,
     )
     return id.String(), err
   }
   ```

---

#### Этап 6: Оптимизация производительности (1 день)

1. **Пул контейнеров**:
   ```go
   // internal/app/executor/pool.go
   type ContainerPool struct {
     available chan *docker.Container
     maxSize   int
   }
   
   func (p *ContainerPool) Get() (*docker.Container, error) {
     select {
     case container := <-p.available:
       return container, nil
     default:
       return p.createNewContainer()
     }
   }
   ```

2. **Кеширование результатов**:
   ```go
   func (e *CodeExecutor) Run(code string) (string, error) {
     cacheKey := fmt.Sprintf("exec:%s", sha256.Sum256([]byte(code)))
     if cached, err := e.cache.Get(cacheKey); err == nil {
       return cached, nil
     }
     // ... выполнение кода
     e.cache.Set(cacheKey, output, 10*time.Minute)
   }
   ```

---

#### Этап 7: Тестирование (2 дня)

1. **Интеграционные тесты**:
   ```go
   func TestGoCodeExecution(t *testing.T) {
     executor := NewExecutor()
     output, err := executor.Run(`package main; func main() { println("OK") }`, "go")
     assert.NoError(t, err)
     assert.Contains(t, output, "OK")
   }
   ```

2. **Тест безопасности**:
   ```go
   func TestDangerousCode(t *testing.T) {
     output, err := executor.Run(`package main; import "os"; func main() { os.Exit(1) }`, "go")
     assert.Error(t, err)
     assert.Contains(t, err.Error(), "permission denied")
   }
   ```

---

#### Этап 8: Развёртывание (1 день)

1. **Dockerfile для сервера**:
   ```dockerfile
   FROM golang:1.20 as builder
   WORKDIR /app
   COPY . .
   RUN CGO_ENABLED=0 go build -o server ./cmd/server
   
   FROM alpine:latest
   COPY --from=builder /app/server /server
   CMD ["/server"]
   ```

2. **docker-compose.yml**:
   ```yaml
   version: '3.8'
   services:
     app:
       build: .
       ports: ["8080:8080"]
       environment:
         - DOCKER_HOST=tcp://host.docker.internal:2375
         - REDIS_URL=redis://redis:6379
       depends_on:
         - redis
     
     redis:
       image: redis:alpine
   ```

---

#### Этап 9: Мониторинг и логирование (дополнительно)

1. **Интеграция Prometheus**:
   ```go
   import "github.com/prometheus/client_golang/prometheus"
   
   var (
     execRequests = prometheus.NewCounterVec(
       prometheus.CounterOpts{
         Name: "code_exec_requests_total",
         Help: "Total code execution requests",
       },
       []string{"lang", "status"},
     )
   )
   
   func init() {
     prometheus.MustRegister(execRequests)
   }
   ```

2. **Логирование**:
   ```go
   func main() {
     logger := zap.NewExample()
     defer logger.Sync()
     
     // В обработчике
     logger.Info("Code executed", 
       zap.String("lang", lang),
       zap.Duration("duration", elapsed),
     )
   }
   ```

---


### Ключевые моменты безопасности:
1. **Изоляция**:
   - Запуск без привилегий: `User: "nobody"`
   - Read-only файловая система
   - Отключение системных вызовов через seccomp

2. **Валидация кода**:
   ```go
   func validateGoCode(code string) error {
     if strings.Contains(code, "syscall") {
       return errors.New("forbidden system call")
     }
     if len(code) > 100000 {
       return errors.New("code too large")
     }
     return nil
   }
   ```

3. **Анти-DDOS**:
   - Ограничение 1 запрос/сек с IP
   - Максимум 5 параллельных выполнений

---

### Оптимизации для продакшена:
1. **Предварительный билд образов**:
   ```bash
   docker build -t go-executor -f Dockerfile.executor .
   ```

2. **Гранулярное управление ресурсами**:
   ```go
   hostConfig.Resources = container.Resources{
     Memory:   calculateMemory(lang),
     CPUQuota: calculateCPU(lang),
   }
   ```

3. **WebSocket для потокового вывода**:
   ```go
   func (c *Client) StreamOutput(ctx context.Context, containerID string) {
     for {
       select {
       case <-ctx.Done(): return
       default:
         output := getNextOutput(containerID)
         c.conn.WriteJSON(output)
       }
     }
   }
   ```

Этот план обеспечивает комплексную реализацию с акцентом на безопасность, производительность и масштабируемость. Для enterprise-решений рекомендуется добавить аутентификацию через JWT и распределённое кеширование.
