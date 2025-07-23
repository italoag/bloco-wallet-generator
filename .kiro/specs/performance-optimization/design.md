# Design Document

## Overview

Este documento descreve o design para otimizar o gerador de carteiras Ethereum bloco-eth para máxima performance através de paralelização multi-thread, otimizações criptográficas e melhorias de gerenciamento de memória. O objetivo é maximizar o throughput de geração de carteiras utilizando todos os recursos disponíveis do processador.

## Architecture

### Current Architecture Analysis

O código atual utiliza uma abordagem single-threaded sequencial:
1. Loop infinito gerando carteiras uma por vez
2. Cada iteração: gera chave privada → deriva endereço → valida padrão
3. Estatísticas atualizadas periodicamente
4. Para quando encontra match

### Proposed Multi-threaded Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Main Controller                          │
├─────────────────────────────────────────────────────────────┤
│ • Detecta CPUs disponíveis                                  │
│ • Cria worker pool                                          │
│ • Coordena workers via channels                             │
│ • Agrega estatísticas                                       │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   Worker Pool Manager                       │
├─────────────────────────────────────────────────────────────┤
│ • Gerencia N workers (N = número de CPUs)                   │
│ • Distribui trabalho via work channels                      │
│ • Coleta resultados via result channels                     │
│ • Implementa graceful shutdown                              │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│              Worker Threads (1 per CPU core)                │
├─────────────────────────────────────────────────────────────┤
│ Worker 1    │ Worker 2    │ Worker 3    │ ... │ Worker N    │
│ • Crypto    │ • Crypto    │ • Crypto    │     │ • Crypto    │
│   pools     │   pools     │   pools     │     │   pools     │
│ • Local     │ • Local     │ • Local     │     │ • Local     │
│   stats     │   stats     │   stats     │     │   stats     │
│ • Generate  │ • Generate  │ • Generate  │     │ • Generate  │
│   wallets   │   wallets   │   wallets   │     │   wallets   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                 Shared Resources                            │
├─────────────────────────────────────────────────────────────┤
│ • Thread-safe statistics aggregator                         │
│ • Result channel for successful matches                     │
│ • Progress display coordinator                              │
│ • Graceful shutdown signal                                  │
└─────────────────────────────────────────────────────────────┘
```

## Components and Interfaces

### 1. WorkerPool

```go
type WorkerPool struct {
    numWorkers    int
    workers       []*Worker
    workChan      chan WorkItem
    resultChan    chan WorkResult
    statsChan     chan WorkerStats
    shutdownChan  chan struct{}
    wg            sync.WaitGroup
}

type WorkItem struct {
    Prefix     string
    Suffix     string
    IsChecksum bool
    BatchSize  int
}

type WorkResult struct {
    Wallet    *Wallet
    Attempts  int64
    WorkerID  int
    Found     bool
    Error     error
}
```

### 2. Worker

```go
type Worker struct {
    id           int
    workChan     chan WorkItem
    resultChan   chan WorkResult
    statsChan    chan WorkerStats
    shutdownChan chan struct{}
    
    // Optimized crypto resources
    cryptoPool   *CryptoPool
    hasherPool   *HasherPool
    bufferPool   *BufferPool
    
    // Local statistics
    localStats   WorkerStats
}

type WorkerStats struct {
    WorkerID      int
    Attempts      int64
    Speed         float64
    LastUpdate    time.Time
}
```

### 3. CryptoPool (Object Pooling)

```go
type CryptoPool struct {
    privateKeyPool sync.Pool
    publicKeyPool  sync.Pool
    bigIntPool     sync.Pool
}

type HasherPool struct {
    keccakPool sync.Pool
}

type BufferPool struct {
    byteBufferPool   sync.Pool
    stringBuilderPool sync.Pool
}
```

### 4. Thread-Safe Statistics Manager

```go
type StatsManager struct {
    mu              sync.RWMutex
    totalAttempts   int64
    workerStats     map[int]WorkerStats
    startTime       time.Time
    lastUpdate      time.Time
    
    // Aggregated metrics
    totalSpeed      float64
    avgSpeed        float64
    peakSpeed       float64
}
```

### 5. Enhanced CLI Interface

```go
// Adicionar nova flag para controle de threads
var threadsFlag int

// Modificar comandos existentes para suportar paralelização
func init() {
    rootCmd.Flags().IntVarP(&threadsFlag, "threads", "t", 0, 
        "Number of threads to use (0 = auto-detect)")
}
```

## Data Models

### Enhanced Wallet Generation Flow

```go
type ParallelWalletGenerator struct {
    config       GenerationConfig
    workerPool   *WorkerPool
    statsManager *StatsManager
    resultChan   chan *WalletResult
    done         chan struct{}
}

type GenerationConfig struct {
    Prefix       string
    Suffix       string
    IsChecksum   bool
    NumThreads   int
    BatchSize    int
    ShowProgress bool
}
```

### Optimized Crypto Operations

```go
// Reutilização de objetos criptográficos
type OptimizedCrypto struct {
    privateKeyBytes [32]byte
    publicKeyBytes  [64]byte
    addressBytes    [20]byte
    hashBytes       [32]byte
    
    // Pré-alocados para evitar GC pressure
    hexBuffer       [64]byte
    stringBuilder   strings.Builder
}
```

## Error Handling

### Thread-Safe Error Management

1. **Worker Error Isolation**: Erros em um worker não afetam outros
2. **Graceful Degradation**: Se um worker falha, outros continuam
3. **Error Aggregation**: Coleta e reporta erros de todos os workers
4. **Timeout Handling**: Implementa timeouts para evitar workers travados

```go
type ErrorManager struct {
    mu     sync.Mutex
    errors []WorkerError
}

type WorkerError struct {
    WorkerID  int
    Error     error
    Timestamp time.Time
    Attempts  int64
}
```

## Testing Strategy

### 1. Unit Tests

- **CryptoPool Tests**: Verificar reutilização correta de objetos
- **Worker Tests**: Testar geração isolada de carteiras
- **StatsManager Tests**: Verificar agregação thread-safe de estatísticas
- **BufferPool Tests**: Validar gerenciamento de memória

### 2. Integration Tests

- **WorkerPool Integration**: Testar coordenação entre workers
- **End-to-End Parallel Generation**: Validar geração completa multi-thread
- **Performance Regression Tests**: Garantir que otimizações não quebram funcionalidade

### 3. Performance Tests

- **Throughput Benchmarks**: Medir addr/s single vs multi-thread
- **Scalability Tests**: Testar performance com diferentes números de threads
- **Memory Usage Tests**: Verificar consumo de RAM com paralelização
- **CPU Utilization Tests**: Medir eficiência de uso de CPU

### 4. Stress Tests

- **Long-running Generation**: Testar estabilidade em execuções longas
- **High Difficulty Patterns**: Testar com padrões muito difíceis
- **Resource Exhaustion**: Testar comportamento com recursos limitados

## Performance Optimizations

### 1. Crypto Optimizations

```go
// Pool de objetos criptográficos reutilizáveis
var (
    privateKeyPool = sync.Pool{
        New: func() interface{} {
            return make([]byte, 32)
        },
    }
    
    keccakPool = sync.Pool{
        New: func() interface{} {
            return sha3.NewLegacyKeccak256()
        },
    }
)
```

### 2. Memory Optimizations

- **Object Pooling**: Reutilizar estruturas criptográficas
- **Buffer Reuse**: Pools para buffers de bytes e strings
- **Reduced Allocations**: Minimizar new() calls no hot path
- **GC Tuning**: Configurar GOGC para reduzir pause times

### 3. Algorithm Optimizations

- **Batch Processing**: Workers processam lotes de tentativas
- **Early Termination**: Para todos os workers quando encontra match
- **Load Balancing**: Distribui trabalho uniformemente
- **Cache Locality**: Otimizar acesso a dados frequentes

### 4. I/O Optimizations

- **Buffered Output**: Agrupar updates de progresso
- **Async Statistics**: Atualizar estatísticas em background
- **Reduced System Calls**: Minimizar chamadas de sistema

## Implementation Phases

### Phase 1: Core Parallelization
- Implementar WorkerPool básico
- Criar Workers com geração paralela
- Thread-safe statistics aggregation
- Basic error handling

### Phase 2: Crypto Optimizations
- Implementar object pooling para crypto
- Otimizar operações de hash
- Reduzir memory allocations
- Buffer reuse patterns

### Phase 3: Advanced Features
- Dynamic thread scaling
- Advanced load balancing
- Performance monitoring
- Memory usage optimization

### Phase 4: Polish & Testing
- Comprehensive test suite
- Performance benchmarking
- Documentation updates
- CLI enhancements

## Monitoring and Metrics

### Real-time Metrics

```go
type PerformanceMetrics struct {
    ThreadUtilization  map[int]float64  // % utilização por thread
    MemoryUsage       int64            // Bytes em uso
    GCPauses          []time.Duration  // Pausas do garbage collector
    TotalThroughput   float64          // addr/s total
    PerThreadSpeed    map[int]float64  // addr/s por thread
    EfficiencyRatio   float64          // Speedup vs single-thread
}
```

### Performance Dashboard

- CPU utilization per core
- Memory usage trends
- Throughput over time
- Thread efficiency metrics
- GC impact analysis

## Security Considerations

1. **Crypto Security**: Manter qualidade criptográfica com paralelização
2. **Random Number Generation**: Garantir entropia adequada em todas as threads
3. **Memory Security**: Limpar buffers sensíveis após uso
4. **Thread Safety**: Evitar race conditions em operações críticas

## Backward Compatibility

- Manter interface CLI existente
- Preservar formato de saída
- Suportar flags existentes
- Graceful fallback para single-thread se necessário