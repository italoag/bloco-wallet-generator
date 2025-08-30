# Design Document

## Overview

Esta funcionalidade estende o bloco-wallet para gerar automaticamente arquivos KeyStore V3 compatíveis com Ethereum para cada carteira gerada. O sistema implementará geração segura de senhas, criptografia de chaves privadas usando padrões estabelecidos, e gerenciamento de arquivos com configurações flexíveis.

## Architecture

### High-Level Components

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   CLI Handler   │───▶│  KeyStore Gen    │───▶│  File Manager   │
│                 │    │     Service      │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │                          │
                              ▼                          ▼
                       ┌──────────────────┐    ┌─────────────────┐
                       │ Password Gen     │    │   File System   │
                       │    Service       │    │                 │
                       └──────────────────┘    └─────────────────┘
```

### Integration Points

- **Wallet Generation**: Integra com o processo existente de geração de carteiras
- **Progress System**: Utiliza o sistema de progresso existente para feedback
- **CLI Framework**: Adiciona novas flags ao comando existente usando Cobra
- **Error Handling**: Integra com o sistema de tratamento de erros existente

## Components and Interfaces

### KeyStore Generator Service

```go
type KeyStoreService struct {
    outputDir    string
    enabled      bool
    passwordGen  *PasswordGenerator
}

type KeyStoreConfig struct {
    OutputDirectory string
    Enabled         bool
    Cipher          string // "aes-128-ctr"
    KDF             string // "scrypt" or "pbkdf2"
}

func NewKeyStoreService(config KeyStoreConfig) *KeyStoreService
func (ks *KeyStoreService) GenerateKeyStore(wallet *Wallet, password string) error
func (ks *KeyStoreService) SaveKeyStoreFiles(address, keystore, password string) error
```

### Password Generator Service

```go
type PasswordGenerator struct {
    minLength int
    charset   PasswordCharset
}

type PasswordCharset struct {
    Lowercase string
    Uppercase string
    Numbers   string
    Special   string
}

func NewPasswordGenerator() *PasswordGenerator
func (pg *PasswordGenerator) GenerateSecurePassword() (string, error)
func (pg *PasswordGenerator) ValidatePassword(password string) error
```

### KeyStore V3 Structure

```go
type KeyStoreV3 struct {
    Address string                 `json:"address"`
    Crypto  KeyStoreCrypto        `json:"crypto"`
    ID      string                `json:"id"`
    Version int                   `json:"version"`
}

type KeyStoreCrypto struct {
    Cipher       string            `json:"cipher"`
    CipherText   string            `json:"ciphertext"`
    CipherParams CipherParams      `json:"cipherparams"`
    KDF          string            `json:"kdf"`
    KDFParams    interface{}       `json:"kdfparams"`
    MAC          string            `json:"mac"`
}
```

## Data Models

### Enhanced Wallet Structure

```go
type Wallet struct {
    PrivateKey string
    Address    string
    // Novos campos para KeyStore
    KeyStore   *KeyStoreV3 `json:"keystore,omitempty"`
    Password   string      `json:"-"` // Nunca serializar
}

type WalletGenerationResult struct {
    Wallet       *Wallet
    KeyStorePath string
    PasswordPath string
    Error        error
}
```

### Configuration Extensions

```go
type Config struct {
    // Campos existentes...
    
    // Novos campos para KeyStore
    KeyStoreEnabled    bool   `json:"keystore_enabled"`
    KeyStoreOutputDir  string `json:"keystore_output_dir"`
    KeyStoreKDF        string `json:"keystore_kdf"`
}
```

## Error Handling

### Error Types

```go
var (
    ErrKeyStoreGeneration = errors.New("failed to generate keystore")
    ErrPasswordGeneration = errors.New("failed to generate password")
    ErrFileCreation      = errors.New("failed to create keystore files")
    ErrDirectoryCreation = errors.New("failed to create output directory")
    ErrInvalidPassword   = errors.New("password does not meet complexity requirements")
)

type KeyStoreError struct {
    Operation string
    Address   string
    Err       error
}

func (e *KeyStoreError) Error() string {
    return fmt.Sprintf("keystore %s failed for address %s: %v", e.Operation, e.Address, e.Err)
}
```

### Error Recovery

- **File System Errors**: Retry com backoff exponencial
- **Permission Errors**: Sugerir soluções ao usuário
- **Disk Space**: Verificar espaço disponível antes da criação
- **Concurrent Access**: Usar locks para evitar conflitos

## Implementation Strategy

### Phase 1: Core KeyStore Generation

1. **Password Generation Service**
   - Implementar gerador de senhas seguras
   - Validação de complexidade
   - Testes unitários para entropia

2. **KeyStore V3 Implementation**
   - Estruturas de dados JSON
   - Algoritmos de criptografia (scrypt/pbkdf2)
   - Geração de MAC para integridade

### Phase 2: File Management

1. **File Operations**
   - Criação de diretórios
   - Escrita atômica de arquivos
   - Tratamento de permissões

2. **Integration with Existing Flow**
   - Modificar função de geração de carteiras
   - Adicionar hooks no processo existente
   - Manter compatibilidade com código atual

### Phase 3: CLI Integration

1. **Command Line Flags**
   - `--keystore-dir`: Diretório de saída
   - `--no-keystore`: Desabilitar geração
   - `--keystore-kdf`: Algoritmo KDF (scrypt/pbkdf2)

2. **Progress Integration**
   - Atualizar barras de progresso
   - Logging de operações de arquivo
   - Estatísticas de arquivos criados

## Testing Strategy

### Unit Tests

```go
func TestPasswordGenerator_GenerateSecurePassword(t *testing.T)
func TestPasswordGenerator_ValidateComplexity(t *testing.T)
func TestKeyStoreService_GenerateKeyStore(t *testing.T)
func TestKeyStoreService_SaveFiles(t *testing.T)
```

### Integration Tests

```go
func TestEndToEndKeyStoreGeneration(t *testing.T)
func TestKeyStoreCompatibility(t *testing.T) // Testar com geth, metamask
func TestConcurrentKeyStoreGeneration(t *testing.T)
```

### Performance Tests

```go
func BenchmarkKeyStoreGeneration(b *testing.B)
func BenchmarkPasswordGeneration(b *testing.B)
func TestKeyStoreGenerationMemoryUsage(t *testing.T)
```

### Security Tests

```go
func TestPasswordEntropy(t *testing.T)
func TestKeyStoreEncryption(t *testing.T)
func TestPrivateKeyNeverExposed(t *testing.T)
```

## Security Considerations

### Password Security

- **Entropia**: Mínimo 128 bits de entropia
- **Geração**: Usar crypto/rand exclusivamente
- **Armazenamento**: Arquivos .pwd com permissões 600
- **Limpeza**: Limpar senhas da memória após uso

### KeyStore Security

- **Algoritmos**: Suporte para scrypt (padrão) e pbkdf2
- **Parâmetros**: Usar parâmetros seguros (N=262144, r=8, p=1 para scrypt)
- **MAC**: Verificação de integridade com HMAC-SHA256
- **IV/Salt**: Geração aleatória para cada keystore

### File Security

- **Permissões**: Arquivos keystore com permissões 600
- **Atomic Writes**: Evitar corrupção durante escrita
- **Backup**: Não sobrescrever arquivos existentes sem confirmação

## Performance Considerations

### Memory Management

- **Object Pooling**: Reutilizar objetos de criptografia
- **Streaming**: Processar arquivos grandes em chunks
- **Garbage Collection**: Minimizar alocações desnecessárias

### I/O Optimization

- **Batch Operations**: Agrupar operações de arquivo
- **Async I/O**: Operações não-bloqueantes quando possível
- **Buffer Management**: Usar buffers apropriados para escrita

### Scalability

- **Concurrent Generation**: Suporte para geração paralela
- **Rate Limiting**: Evitar sobrecarga do sistema de arquivos
- **Progress Tracking**: Atualizações eficientes de progresso