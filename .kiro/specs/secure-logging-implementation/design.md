# Design Document

## Overview

Este documento descreve o design de um sistema de logging seguro para o bloco-wallet que substitui a implementação atual que registra informações sensíveis. O novo sistema implementará logging estruturado com diferentes níveis de severidade, controle granular de configuração e garantia de que nenhuma informação criptográfica sensível seja registrada.

## Architecture

### Core Components

1. **SecureLogger**: Substitui o WalletLogger atual com implementação segura
2. **LogLevel**: Enum para controle de níveis de logging (ERROR, WARN, INFO, DEBUG)
3. **LogConfig**: Estrutura de configuração para controle de logging
4. **LogEntry**: Estrutura padronizada para entradas de log
5. **LogFormatter**: Interface para formatação de logs (JSON, texto estruturado)

### Logging Levels

- **ERROR**: Erros críticos que impedem operação
- **WARN**: Avisos sobre situações não ideais mas não críticas  
- **INFO**: Informações operacionais importantes (padrão)
- **DEBUG**: Informações detalhadas para debugging (sem dados sensíveis)

### Security Principles

1. **Zero Sensitive Data**: Nunca registrar chaves privadas, públicas ou seeds
2. **Minimal Data**: Registrar apenas o mínimo necessário para operação e debugging
3. **Structured Logging**: Usar formato estruturado para facilitar análise
4. **Configurable**: Permitir desabilitar logging completamente se necessário

## Components and Interfaces

### SecureLogger Interface

```go
type SecureLogger interface {
    Error(msg string, fields ...LogField) error
    Warn(msg string, fields ...LogField) error  
    Info(msg string, fields ...LogField) error
    Debug(msg string, fields ...LogField) error
    
    LogWalletGenerated(address string, attempts int, duration time.Duration, threadID int) error
    LogOperationStart(operation string, params map[string]interface{}) error
    LogOperationComplete(operation string, stats OperationStats) error
    LogError(operation string, err error, context map[string]interface{}) error
    
    Close() error
    SetLevel(level LogLevel) error
    IsEnabled(level LogLevel) bool
}
```

### LogConfig Structure

```go
type LogConfig struct {
    Enabled     bool      // Enable/disable logging completely
    Level       LogLevel  // Minimum log level to record
    Format      LogFormat // JSON, TEXT, or STRUCTURED
    OutputFile  string    // Log file path (empty for stdout)
    MaxFileSize int64     // Max log file size in bytes
    MaxFiles    int       // Max number of rotated files
    BufferSize  int       // Buffer size for async logging
}
```

### LogEntry Structure

```go
type LogEntry struct {
    Timestamp   time.Time              `json:"timestamp"`
    Level       LogLevel               `json:"level"`
    Message     string                 `json:"message"`
    Operation   string                 `json:"operation,omitempty"`
    ThreadID    int                    `json:"thread_id,omitempty"`
    Fields      map[string]interface{} `json:"fields,omitempty"`
    Error       string                 `json:"error,omitempty"`
}
```

### Safe Data Types

Apenas os seguintes tipos de dados são permitidos nos logs:

**Wallet Generation:**
- Address (endereço público Ethereum)
- Attempt count (número de tentativas)
- Duration (tempo de geração)
- Thread ID (identificador da thread)
- Success/failure status

**Operations:**
- Operation name/type
- Start/end timestamps
- Duration
- Success/failure status
- Error messages (sanitized)
- Performance metrics

**System:**
- Configuration parameters (non-sensitive)
- Resource usage metrics
- Thread/worker status
- Progress indicators

## Data Models

### Wallet Generation Log Entry

```go
type WalletGenerationEntry struct {
    Address   string        `json:"address"`
    Attempts  int           `json:"attempts"`
    Duration  time.Duration `json:"duration_ns"`
    ThreadID  int           `json:"thread_id"`
    Pattern   string        `json:"pattern,omitempty"`
    Timestamp time.Time     `json:"timestamp"`
}
```

### Operation Log Entry

```go
type OperationEntry struct {
    Operation string                 `json:"operation"`
    Status    string                 `json:"status"` // started, completed, failed
    Duration  time.Duration          `json:"duration_ns,omitempty"`
    Params    map[string]interface{} `json:"params,omitempty"`
    Stats     *OperationStats        `json:"stats,omitempty"`
    Error     string                 `json:"error,omitempty"`
    Timestamp time.Time              `json:"timestamp"`
}
```

### Performance Metrics

```go
type PerformanceMetrics struct {
    WalletsPerSecond float64 `json:"wallets_per_second"`
    TotalWallets     int64   `json:"total_wallets"`
    TotalAttempts    int64   `json:"total_attempts"`
    AverageDuration  float64 `json:"avg_duration_ms"`
    ThreadCount      int     `json:"thread_count"`
    CPUUsage         float64 `json:"cpu_usage_percent,omitempty"`
    MemoryUsage      int64   `json:"memory_usage_bytes,omitempty"`
}
```

## Error Handling

### Error Sanitization

Todos os erros são sanitizados antes do logging para remover informações sensíveis:

1. **Crypto Errors**: Remover dados de chaves ou seeds dos erros
2. **Validation Errors**: Registrar tipo de erro sem dados de entrada
3. **File Errors**: Registrar paths relativos, não absolutos
4. **Network Errors**: Remover credenciais ou tokens de URLs

### Error Categories

```go
type ErrorCategory string

const (
    ErrorCrypto     ErrorCategory = "crypto"
    ErrorValidation ErrorCategory = "validation" 
    ErrorIO         ErrorCategory = "io"
    ErrorNetwork    ErrorCategory = "network"
    ErrorSystem     ErrorCategory = "system"
)
```

### Fallback Behavior

Se o logging falhar:
1. Continuar operação normalmente
2. Emitir warning para stderr (uma vez)
3. Desabilitar logging automaticamente
4. Não interromper geração de wallets

## Testing Strategy

### Unit Tests

1. **SecureLogger Tests**
   - Verificar que dados sensíveis nunca são registrados
   - Testar diferentes níveis de log
   - Validar formatação de saída
   - Testar configuração e desabilitação

2. **Data Sanitization Tests**
   - Testar remoção de chaves privadas de erros
   - Validar sanitização de URLs e paths
   - Verificar limpeza de dados de entrada

3. **Performance Tests**
   - Medir overhead do logging
   - Testar logging assíncrono
   - Validar rotação de arquivos

### Integration Tests

1. **End-to-End Logging**
   - Executar geração completa de wallets
   - Verificar logs gerados
   - Confirmar ausência de dados sensíveis

2. **Error Scenarios**
   - Simular falhas de I/O
   - Testar comportamento com disco cheio
   - Validar fallback para stdout

### Security Tests

1. **Sensitive Data Detection**
   - Scan automático de logs por padrões de chaves
   - Verificação de vazamento de dados
   - Auditoria de campos registrados

2. **Configuration Security**
   - Testar desabilitação completa
   - Validar controle de níveis
   - Verificar permissões de arquivos

## Implementation Notes

### Migration Strategy

1. **Phase 1**: Implementar SecureLogger mantendo interface compatível
2. **Phase 2**: Substituir WalletLogger por SecureLogger no pool
3. **Phase 3**: Adicionar configuração CLI para controle de logging
4. **Phase 4**: Remover implementação antiga e arquivos de log existentes

### Backward Compatibility

- Manter arquivos de log antigos como estão (não modificar)
- Novos logs usarão formato seguro
- Adicionar warning sobre logs antigos contendo dados sensíveis
- Fornecer script para limpeza de logs antigos (opcional)

### Performance Considerations

- Usar buffering assíncrono para minimizar impacto
- Implementar rate limiting para evitar spam de logs
- Otimizar serialização JSON para performance
- Permitir desabilitação completa para máxima performance