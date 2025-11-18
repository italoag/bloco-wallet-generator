# Issue: Multi-chain wallet generation support (Bitcoin & Solana)

## Context
O projeto atualmente cobre apenas o fluxo de geração de carteiras Ethereum. As dependências, validações e formatos esperados estão profundamente acoplados ao EVM, tornando o código difícil de estender para outras redes solicitadas pelos usuários.

## Problema
1. **Worker e geração de carteiras acoplados ao Ethereum**: uso direto de `go-ethereum/crypto`, pressupõe curvas secp256k1 e endereços de 20 bytes. Nenhuma forma de injetar geradores/formatadores específicos por rede.
2. **CLI e validação restritos a hex + EIP-55**: flags e validações só aceitam prefixos/sufixos hexadecimais e aplicam checksum Ethereum, impedindo padrões Base58 necessários para Bitcoin/Solana.
3. **Configuração sem noção de rede**: `internal/config` não possui campo para selecionar redes nem parâmetros por rede (derivation path, algoritmos, formatos), o que impede habilitar futuras integrações de forma declarativa.

## Impacto
- Sem suporte multi-chain não é possível atender pedidos de carteiras Bitcoin ou Solana.
- Cada nova rede exigiria fork do worker/CLI, reduzindo mantenibilidade.
- Experiência do usuário limitada, pois o CLI não comunica opções de redes ou diferenças de validação.

## Critérios de aceite
- Introduzir abstrações para geração de carteiras por rede (ex.: `ChainAdapter`).
- CLI deve permitir selecionar a rede (`--chain`) e validar padrões conforme a rede escolhida.
- Configuração deve permitir definir parâmetros específicos por rede.
- Documentação atualizada descrevendo como habilitar Bitcoin e Solana.

## Referências
- Solicitação do usuário por carteiras Bitcoin e Solana.
- Arquitetura atual concentrada em `internal/worker`, `pkg/wallet`, `cmd/bloco-eth`.
