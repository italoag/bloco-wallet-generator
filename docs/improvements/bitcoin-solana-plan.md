# Plano detalhado: suporte a carteiras Bitcoin e Solana

Este documento detalha o plano de melhoria para estender o gerador de carteiras `bloco-wallet-generator` além da rede Ethereum, cobrindo requisitos arquiteturais, ajustes de CLI/config e planos de implementação específicos para Bitcoin e Solana.

## 1. Desacoplar geração por rede
- Definir interface `ChainAdapter` responsável por `GenerateKeyMaterial`, `FormatAddress`, `MatchesPattern` e `DescribeConstraints`.
- Extrair implementação atual para `ethereumAdapter`, mantendo dependências de `go-ethereum/crypto` e checksum EIP-55.
- Atualizar `internal/worker` para receber o adapter injetado via config e remover o uso direto de `crypto.PubkeyToAddress`.
- Estender `pkg/wallet.Wallet` com campo `Chain` e formatos específicos (hex/Base58) delegando validação ao adapter.
- Cobrir com testes unitários garantindo que o worker consome o adapter selecionado.

## 2. CLI e validação sensíveis à rede
- Adicionar flag `--chain` (eth/btc/sol) em `cmd/bloco-eth` e propagar para `internal/config`.
- Tornar `GenerationCriteria` agnóstico, validando prefixo/sufixo através do adapter escolhido (hex para ETH, Base58 para BTC/SOL).
- Atualizar ajuda do CLI descrevendo padrões por rede e exemplos de uso.
- Alinhar comandos auxiliares (`stats`, `benchmark`) com o mesmo seletor de rede.

## 3. Configuração por rede
- Expandir `internal/config.Config` com `Chain string` e mapa `Networks map[string]NetworkConfig`.
- `NetworkConfig` deve conter: curva, encoding, derivation path padrão, versão de endereço, opções específicas (checksum, WIF, etc.).
- Carregar padrões em `DefaultConfig` (Ethereum) e permitir sobrescrita via variáveis de ambiente (`BLOCO_CHAIN`, `BLOCO_NETWORK_*`).
- Passar o bloco selecionado para o adapter correspondente.

## 4. Implementação para Bitcoin
1. Criar pacote `internal/crypto/bitcoin`:
   - Gerar seeds com BIP-39 e derivar chaves usando BIP-32/BIP-44 (`m/44'/0'/0'/0/0`).
   - Suportar endereços P2PKH usando Base58Check (versão 0x00 mainnet).
   - Exportar private key também em WIF para facilitar importação.
2. Atualizar `GenerationCriteria` para permitir padrões Base58 usando funções utilitárias do adapter.
3. Integrar o adapter no worker, reaproveitando o loop de tentativa/erro.
4. Adicionar testes cobrindo conversão chave privada → endereço Base58Check e validação de prefixos/sufixos.
5. Documentar no README como habilitar `--chain btc`, ressaltando custo computacional de Base58.

## 5. Implementação para Solana
1. Criar pacote `internal/crypto/solana` baseado em `crypto/ed25519`:
   - Gerar seeds de 32 bytes e produzir pares `ed25519.PrivateKey/PublicKey`.
   - Formatar endereços como Base58 (valor da public key) e permitir exportar secret key em Base64/Base58 compatível com CLI oficial.
2. Ajustar worker para lidar com ed25519 (sem `crypto.S256`) mantendo paralelismo.
3. Suportar derivação `m/44'/501'/0'/0'` reaproveitando BIP-39 + libs específicas para ed25519.
4. Atualizar validação de padrões para Base58 case-sensitive.
5. Criar testes determinísticos com seeds conhecidos e exemplos oficiais da Solana.
6. Atualizar documentação descrevendo uso de `--chain sol` e exemplos de prefixos.

## 6. Documentação e comunicação
- Atualizar README com tabela comparando redes suportadas, formatos de endereço e flags relevantes.
- Incluir exemplos completos de comandos para ETH, BTC e SOL.
- Comunicar limitações (ex.: suporte inicial apenas para P2PKH, necessidade de libs específicas para ed25519).

Este plano serve tanto como especificação técnica quanto base para abertura de PRs e acompanhamento no GitHub.
