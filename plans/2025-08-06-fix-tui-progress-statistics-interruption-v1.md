# Correção da Interrupção Prematura da Barra de Progresso e Estatísticas na TUI

## Objetivo
Corrigir o problema onde a barra de progresso e estatísticas param de ser atualizadas prematuramente quando a última carteira é gerada, garantindo que as informações estatísticas sejam totalmente atualizadas e visíveis ao usuário antes da finalização.

## Implementação Plan

1. **Análise Detalhada do Fluxo de Dados**
   - Dependências: Nenhuma
   - Notas: Examinar o fluxo completo de dados entre workers, stats manager e TUI para identificar pontos de falha
   - Arquivos: `main.go:generateWalletsInBackground`, `tui/progress.go`, `stats_manager.go`
   - Status: Não Iniciado

2. **Identificação dos Pontos de Fechamento Prematuro dos Canais**
   - Dependências: Tarefa 1
   - Notas: Mapear exatamente onde `walletResultsChan` e `statsUpdateChan` são fechados prematuramente na linha `main.go:2196-2201`
   - Arquivos: `main.go:2196-2201`, `main.go:2238-2267`
   - Status: Não Iniciado

3. **Implementação de Sincronização de Finalização**
   - Dependências: Tarefa 2
   - Notas: Adicionar canal `completionChan` para sincronizar o fechamento entre geração e processamento da TUI
   - Arquivos: `main.go:generateWalletsInBackground`, `main.go:displayMultipleWalletsTUI`
   - Status: Não Iniciado

4. **Correção do Ticker de Estatísticas Finais**
   - Dependências: Tarefa 3
   - Notas: Modificar o ticker na linha `main.go:2128-2139` para continuar atualizando estatísticas até confirmação de processamento completo
   - Arquivos: `main.go:2128-2139`
   - Status: Não Iniciado

5. **Implementação de Período de Visualização Final**
   - Dependências: Tarefa 4
   - Notas: Adicionar delay configurável para permitir visualização das estatísticas finais antes do fechamento
   - Arquivos: `tui/progress.go:Update`, `main.go:displayMultipleWalletsTUI`
   - Status: Não Iniciado

6. **Correção da Goroutine de Processamento de Resultados**
   - Dependências: Tarefa 5
   - Notas: Modificar a goroutine na linha `main.go:2238-2267` para processar completamente antes de sinalizar término
   - Arquivos: `main.go:2238-2267`
   - Status: Não Iniciado

7. **Adição de Estado de Finalização na TUI**
   - Dependências: Tarefa 6
   - Notas: Implementar estado de "finalização" no modelo de progresso para mostrar estatísticas completas
   - Arquivos: `tui/progress.go:ProgressModel`, `tui/progress.go:Update`
   - Status: Não Iniciado

8. **Testes de Verificação e Validação**
   - Dependências: Tarefa 7
   - Notas: Testar geração de múltiplas carteiras para verificar se estatísticas permanecem atualizadas até o final
   - Arquivos: Todos os arquivos modificados
   - Status: Não Iniciado

## Critérios de Verificação
- A barra de progresso deve permanecer ativa e atualizada até 100% de conclusão
- As estatísticas finais (velocidade total, tempo total, tentativas) devem ser exibidas completamente
- A TUI deve permanecer responsiva durante todo o processo de geração
- Não deve haver race conditions entre fechamento de canais e processamento da TUI
- O usuário deve ter tempo adequado para visualizar as estatísticas finais

## Riscos Potenciais e Mitigações

1. **Race Condition entre Fechamento de Canais e Processamento**
   Mitigação: Implementar sincronização explícita com canal de confirmação de processamento completo

2. **Deadlock por Sincronização Inadequada**
   Mitigação: Usar timeouts em todas as operações de sincronização e canais com buffer adequado

3. **Degradação de Performance por Delay Adicional**
   Mitigação: Implementar delay configurável e otimizar apenas as partes críticas do fluxo

4. **Inconsistência de Estados na TUI**
   Mitigação: Implementar máquina de estados clara com transições bem definidas

5. **Problema em Geração de Carteira Única**
   Mitigação: Verificar e corrigir também o fluxo de geração individual de carteiras

## Abordagens Alternativas

1. **Abordagem por Buffer de Delay**: Implementar buffer temporal simples antes do fechamento dos canais
2. **Abordagem por Confirmação Explícita**: Requerer confirmação da TUI antes de finalizar o processo
3. **Abordagem por Estados Separados**: Separar completamente os estados de "geração" e "finalização" na TUI