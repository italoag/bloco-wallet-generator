# Design Document

## Overview

O problema identificado está na função `isValidBlocoAddress` no arquivo `internal/worker/pool.go`. A validação de sufixo está funcionando corretamente para validação case-insensitive, mas há um bug na validação de checksum EIP-55 onde o sufixo não está sendo validado adequadamente na função `isEIP55Checksum`.

### Root Cause Analysis

Após análise do código, identifiquei que:

1. **Validação Case-Insensitive**: Funciona corretamente - tanto prefix quanto suffix são validados
2. **Validação EIP-55 Checksum**: Tem um bug na função `isEIP55Checksum` onde a validação do sufixo não está sendo executada corretamente

O problema específico está na função `isEIP55Checksum` em `internal/worker/pool.go` linha ~200, onde a lógica de validação do sufixo tem uma falha na comparação de case-sensitivity.

## Architecture

### Current Architecture
```
CLI Command → Worker Pool → isValidBlocoAddress() → isEIP55Checksum() [BUG HERE]
                                                 → Case-insensitive validation [OK]
```

### Fixed Architecture
```
CLI Command → Worker Pool → isValidBlocoAddress() → Fixed isEIP55Checksum()
                                                 → Case-insensitive validation [OK]
```

## Components and Interfaces

### Affected Components

1. **Worker Pool (`internal/worker/pool.go`)**
   - Função `isValidBlocoAddress()` - funciona corretamente
   - Função `isEIP55Checksum()` - **CONTÉM O BUG**

2. **Validation Strategy (`internal/validation/strategy.go`)**
   - Implementações corretas, mas não são usadas pelo worker pool
   - Worker pool usa sua própria implementação inline

### Bug Details

Na função `isEIP55Checksum`, o problema está na validação do sufixo:

```go
// CÓDIGO ATUAL (BUGGY)
if suffix != "" {
    suffixStart := len(checksumAddr) - len(suffix)
    if suffixStart < 2 {
        return false
    }
    suffixPart := checksumAddr[suffixStart:]
    if !strings.EqualFold(suffixPart, suffix) {
        return false
    }
    // Check if the actual case matches the desired pattern
    if suffixPart != suffix {  // ← BUG: Esta comparação sempre falha para checksum
        return false
    }
}
```

O problema é que `suffixPart` contém o sufixo com o case correto do checksum EIP-55, mas `suffix` contém o padrão desejado pelo usuário. A comparação `suffixPart != suffix` sempre falha quando o checksum requer case diferente do especificado pelo usuário.

## Data Models

### Existing Models (No Changes Required)
- `wallet.GenerationCriteria` - já suporta prefix, suffix e checksum
- `wallet.GenerationResult` - não precisa de mudanças
- `wallet.Wallet` - não precisa de mudanças

## Error Handling

### Current Error Handling
- Validação silenciosa (retorna false)
- Sem logs de debug para identificar falhas

### Improved Error Handling
- Manter validação silenciosa para performance
- Adicionar logs de debug opcionais via variável de ambiente
- Manter compatibilidade com código existente

## Testing Strategy

### Test Cases Required

1. **Suffix-only validation (case-insensitive)**
   ```go
   criteria := wallet.GenerationCriteria{
       Suffix: "abc",
       IsChecksum: false,
   }
   // Should generate addresses ending with "abc" (case-insensitive)
   ```

2. **Suffix-only validation (checksum)**
   ```go
   criteria := wallet.GenerationCriteria{
       Suffix: "abc", 
       IsChecksum: true,
   }
   // Should generate addresses ending with "abc" respecting EIP-55 checksum
   ```

3. **Prefix + Suffix validation (case-insensitive)**
   ```go
   criteria := wallet.GenerationCriteria{
       Prefix: "123",
       Suffix: "abc",
       IsChecksum: false,
   }
   // Should generate addresses starting with "123" and ending with "abc"
   ```

4. **Prefix + Suffix validation (checksum)**
   ```go
   criteria := wallet.GenerationCriteria{
       Prefix: "123",
       Suffix: "abc", 
       IsChecksum: true,
   }
   // Should generate addresses with both patterns respecting EIP-55 checksum
   ```

### Performance Testing
- Verificar que a correção não impacta performance
- Benchmark antes e depois da correção
- Validar que a complexidade algorítmica permanece O(1)

### Integration Testing
- Testar com CLI real usando diferentes combinações
- Verificar que testes existentes continuam passando
- Testar casos extremos (suffixes longos, overlapping patterns)

## Implementation Plan

### Phase 1: Fix Core Bug
1. Corrigir a função `isEIP55Checksum` em `internal/worker/pool.go`
2. Implementar lógica correta para validação de sufixo com checksum
3. Manter compatibilidade com código existente

### Phase 2: Add Tests
1. Criar testes unitários para todas as combinações prefix/suffix
2. Adicionar testes de integração via CLI
3. Criar benchmarks para verificar performance

### Phase 3: Validation
1. Executar todos os testes existentes
2. Testar manualmente com CLI
3. Verificar que a correção resolve o problema reportado

## Technical Details

### Correct Suffix Validation Logic

A lógica correta deve:

1. **Para validação case-insensitive**: Comparar lowercase versions (já funciona)
2. **Para validação checksum**: 
   - Extrair o sufixo do endereço gerado
   - Verificar se o sufixo lowercase corresponde ao padrão desejado
   - Verificar se o case do sufixo está correto conforme EIP-55
   - **NÃO** comparar diretamente o case do sufixo com o padrão do usuário

### Fixed Implementation Approach

```go
// IMPLEMENTAÇÃO CORRIGIDA
if suffix != "" {
    suffixStart := len(checksumAddr) - len(suffix)
    if suffixStart < 2 {
        return false
    }
    suffixPart := checksumAddr[suffixStart:]
    
    // First check if the pattern matches (case-insensitive)
    if !strings.EqualFold(suffixPart, suffix) {
        return false
    }
    
    // For checksum validation, we need to verify that the suffix part
    // has the correct case according to EIP-55, not that it matches
    // the user's input case exactly
    // The checksum correctness is already validated by toChecksumAddress()
}
```

## Migration Strategy

### Backward Compatibility
- Manter todas as interfaces existentes
- Não quebrar testes existentes
- Manter performance equivalente

### Rollout Plan
1. Implementar correção
2. Executar testes completos
3. Testar manualmente
4. Deploy da correção

## Monitoring and Validation

### Success Criteria
- Sufixos funcionam corretamente com e sem checksum
- Todos os testes existentes continuam passando
- Performance mantida
- CLI gera carteiras com prefix+suffix corretamente

### Validation Steps
1. Executar `make test` - todos os testes devem passar
2. Testar CLI: `./bloco-eth --prefix abc --suffix def --count 1`
3. Testar CLI com checksum: `./bloco-eth --prefix abc --suffix def --checksum --count 1`
4. Verificar que endereços gerados atendem ambos os critérios