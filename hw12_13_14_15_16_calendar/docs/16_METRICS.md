## Мониторинг календаря

В сервисе календаря реализован endpoint `GET /metrics`, совместимый с Prometheus.
Метрики собираются в приватном `prometheus.Registry` и не используют глобальное состояние.

### Endpoint

```text
http://localhost:8888/metrics
```

В Docker Compose Prometheus запускается вместе с сервисом и собирает метрики по адресу
`http://calendar:8080/metrics` каждые 5 секунд.

Веб-интерфейс Prometheus:

```text
http://localhost:9090
```

Проверить состояние scrape-цели можно на странице:

```text
http://localhost:9090/targets
```

### Метрики HTTP API

| Метрика | Тип | Метки | Назначение |
|---|---|---|---|
| `calendar_http_requests_total` | counter | `method`, `route`, `status` | Общее количество HTTP-запросов, включая ошибочные ответы. |
| `calendar_http_request_duration_seconds` | histogram | `method`, `route`, `status` | Время обработки HTTP-запросов в секундах. |
| `calendar_http_requests_in_flight` | gauge | — | Количество запросов, обрабатываемых в данный момент. |

В метке `route` используется шаблон маршрута chi, например `/events` или
`/events/{id}`, а не сырой URL. Это ограничивает кардинальность меток.
Запросы к самому `/metrics` не учитываются, чтобы scrape-запросы Prometheus
не искажали статистику API.

### Метрики бизнес-операций

| Метрика | Тип | Метки | Назначение |
|---|---|---|---|
| `calendar_events_created_total` | counter | — | Количество успешно созданных событий. |
| `calendar_events_updated_total` | counter | — | Количество успешно обновлённых событий. |
| `calendar_events_deleted_total` | counter | — | Количество успешно удалённых событий. |
| `calendar_events_listed_total` | counter | — | Суммарное количество событий, возвращённых в ответах на список. |
| `calendar_event_operation_errors_total` | counter | `operation` | Количество ошибок бизнес-операций: `create`, `update`, `delete`, `list`, `list_all`. |

HTTP-метрики показывают состояние API в целом, включая ошибки валидации
и конфликты через метку `status`. Метрика `calendar_event_operation_errors_total`
фиксирует ошибки слоя приложения и хранилища, а бизнес-счётчики позволяют
отдельно отслеживать успешность основных сценариев календаря.

### Примеры запросов Prometheus

Интенсивность запросов:

```promql
sum(rate(calendar_http_requests_total[5m]))
```

Доля ответов с ошибками 5xx:

```promql
sum(rate(calendar_http_requests_total{status=~"5.."}[5m]))
/
sum(rate(calendar_http_requests_total[5m]))
```

95-й процентиль времени обработки по маршрутам:

```promql
histogram_quantile(
  0.95,
  sum by (le, route) (
    rate(calendar_http_request_duration_seconds_bucket[5m])
  )
)
```

Количество созданных событий в минуту:

```promql
increase(calendar_events_created_total[1m])
```

Ошибки бизнес-операций по типам:

```promql
sum by (operation) (
  rate(calendar_event_operation_errors_total[5m])
)
```

### Как использовать метрики

- Рост `calendar_http_requests_total` и `calendar_http_requests_in_flight`
  показывает текущую нагрузку на API.
- `calendar_http_request_duration_seconds` помогает находить медленные
  маршруты и деградацию времени ответа.
- Рост `calendar_http_requests_total{status=~"5.."}` указывает на проблемы
  сервиса или зависимостей.
- `calendar_event_operation_errors_total` позволяет отличить ошибки
  валидации и конфликты от внутренних сбоев бизнес-операций.
- `calendar_events_created_total`, `calendar_events_updated_total`,
  `calendar_events_deleted_total` и `calendar_events_listed_total`
  показывают фактическую активность пользователей.

### Границы текущей реализации

Сервис календаря не содержит фоновых задач. Scheduler и storer — отдельные
процессы, и в выбранном объёме hw16 их собственные `/metrics` не поднимаются.
При необходимости их можно инструментировать отдельными metrics-серверами
и добавить соответствующие scrape-цели в Prometheus.
