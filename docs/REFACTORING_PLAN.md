# Plano de Refatoração Bloco-ETH

## Estrutura de Diretórios Proposta

```
bloco-eth/
├── cmd/
│   └── bloco-eth/
│       └── main.go                 # Entry point minimalista
├── internal/
│   ├── config/
│   │   ├── config.go              # Configuração centralizada
│   │   └── validation.go          # Validação de configuração
│   ├── crypto/
│   │   ├── address.go             # Geração de endereços
│   │   ├── keys.go                # Geração de chaves privadas
│   │   ├── checksum.go            # Validação de checksum
│   │   └── pools.go               # Object pools otimizados
│   ├── validation/
│   │   ├── strategy.go            # Strategy pattern para validação
│   │   ├── checksum.go            # Validação de checksum
│   │   └── pattern.go             # Validação de padrões
│   ├── worker/
│   │   ├── pool.go                # Worker pool gerenciado
│   │   ├── worker.go              # Worker individual
│   │   ├── processor.go           # Processamento de trabalho
│   │   └── stats.go               # Coleta de estatísticas
│   ├── progress/
│   │   ├── manager.go             # Gerenciamento de progresso
│   │   ├── observer.go            # Observer pattern
│   │   └── display.go             # Display de progresso
│   ├── stats/
│   │   ├── manager.go             # Gerenciamento de estatísticas
│   │   ├── calculator.go          # Cálculos estatísticos
│   │   └── metrics.go             # Métricas de performance
│   ├── tui/
│   │   ├── manager.go             # Gerenciamento TUI
│   │   ├── progress.go            # Componente de progresso
│   │   ├── stats.go               # Componente de estatísticas
│   │   ├── styles.go              # Estilos e temas
│   │   └── utils.go               # Utilitários TUI
│   └── cli/
│       ├── commands.go            # Comandos CLI
│       ├── flags.go               # Definição de flags
│       └── handlers.go            # Handlers de comandos
├── pkg/
│   ├── wallet/
│   │   ├── types.go               # Tipos de wallet
│   │   ├── generator.go           # Interface de geração
│   │   └── result.go              # Resultados de geração
│   ├── errors/
│   │   ├── types.go               # Tipos de erro customizados
│   │   └── handlers.go            # Handlers de erro
│   └── utils/
│       ├── hex.go                 # Utilitários hex
│       ├── format.go              # Formatação de números/tempo
│       └── context.go             # Utilitários de contexto
├── test/
│   ├── integration/               # Testes de integração
│   ├── benchmark/                 # Benchmarks
│   └── testdata/                  # Dados de teste
└── docs/
    ├── architecture.md            # Documentação da arquitetura
    └── api.md                     # Documentação da API
```

## Fases de Implementação

### Fase 1: Estrutura Base (1-2 dias)
- [ ] Criar estrutura de diretórios
- [ ] Implementar sistema de configuração
- [ ] Criar tipos básicos e interfaces
- [ ] Implementar sistema de erros

### Fase 2: Camada de Crypto (2-3 dias)
- [ ] Extrair lógica criptográfica
- [ ] Implementar object pools otimizados
- [ ] Criar interfaces para geração de endereços
- [ ] Implementar validação de checksum

### Fase 3: Sistema de Workers (2-3 dias)
- [ ] Refatorar worker pool
- [ ] Implementar processamento adaptativo
- [ ] Criar sistema de estatísticas thread-safe
- [ ] Implementar health monitoring

### Fase 4: Sistema de Progresso (1-2 dias)
- [ ] Implementar observer pattern
- [ ] Refatorar display de progresso
- [ ] Criar sistema de métricas

### Fase 5: Interface TUI (1-2 dias)
- [ ] Refatorar componentes TUI
- [ ] Implementar sistema de estilos
- [ ] Corrigir testes falhando

### Fase 6: CLI e Main (1 dia)
- [ ] Criar main.go minimalista
- [ ] Implementar comandos CLI
- [ ] Integrar todos os componentes

### Fase 7: Testes e Otimização (2-3 dias)
- [ ] Criar testes abrangentes
- [ ] Implementar benchmarks
- [ ] Otimizar performance
- [ ] Documentação

## Critérios de Sucesso
- [ ] main.go < 200 linhas
- [ ] Todos os testes passando
- [ ] Performance mantida ou melhorada
- [ ] Cobertura de testes > 80%
- [ ] Documentação completa