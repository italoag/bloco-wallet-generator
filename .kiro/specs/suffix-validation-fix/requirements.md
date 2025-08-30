# Requirements Document

## Introduction

O sistema de geração de carteiras bloco possui um defeito crítico onde o parâmetro `--suffix` está sendo ignorado durante a validação de endereços. Atualmente, apenas o `--prefix` está sendo respeitado, resultando em carteiras que não atendem aos critérios completos especificados pelo usuário. Este defeito compromete a funcionalidade principal do sistema e precisa ser corrigido para garantir que ambos os parâmetros (prefix e suffix) sejam validados corretamente.

## Requirements

### Requirement 1

**User Story:** Como usuário do bloco-eth, eu quero que quando especificar tanto --prefix quanto --suffix, o sistema gere carteiras que atendam a ambos os critérios simultaneamente, para que eu possa obter endereços com o padrão exato que desejo.

#### Acceptance Criteria

1. WHEN o usuário executa o comando com --prefix "abc" AND --suffix "def" THEN o sistema SHALL gerar apenas endereços que começam com "abc" AND terminam com "def"
2. WHEN o usuário especifica apenas --suffix "xyz" THEN o sistema SHALL gerar endereços que terminam com "xyz" independentemente do prefixo
3. WHEN o usuário especifica apenas --prefix "123" THEN o sistema SHALL gerar endereços que começam com "123" independentemente do sufixo
4. WHEN o usuário especifica tanto --prefix quanto --suffix THEN o sistema SHALL validar ambos os critérios antes de considerar um endereço como válido

### Requirement 2

**User Story:** Como desenvolvedor mantendo o código, eu quero que a função de validação seja clara e testável, para que eu possa garantir que todos os cenários de prefix/suffix funcionem corretamente.

#### Acceptance Criteria

1. WHEN a função de validação é chamada com prefix e suffix THEN ela SHALL verificar ambos os critérios de forma independente
2. WHEN a função de validação encontra um endereço que atende apenas ao prefix THEN ela SHALL rejeitar o endereço se um suffix também foi especificado
3. WHEN a função de validação encontra um endereço que atende apenas ao suffix THEN ela SHALL rejeitar o endereço se um prefix também foi especificado
4. IF apenas um critério (prefix OU suffix) é especificado THEN o sistema SHALL validar apenas esse critério

### Requirement 3

**User Story:** Como usuário, eu quero que o sistema mantenha a mesma performance mesmo quando validando tanto prefix quanto suffix, para que a correção do bug não impacte negativamente a velocidade de geração.

#### Acceptance Criteria

1. WHEN o sistema valida prefix e suffix simultaneamente THEN a performance SHALL ser comparável à validação de apenas um critério
2. WHEN o sistema executa a validação corrigida THEN ela SHALL manter a mesma complexidade algorítmica O(1)
3. WHEN o sistema processa múltiplas validações THEN não SHALL haver degradação significativa de performance

### Requirement 4

**User Story:** Como usuário, eu quero que os testes existentes continuem passando após a correção, para garantir que a funcionalidade existente não seja quebrada.

#### Acceptance Criteria

1. WHEN a correção é implementada THEN todos os testes existentes SHALL continuar passando
2. WHEN novos testes são adicionados para suffix THEN eles SHALL cobrir todos os cenários de combinação prefix/suffix
3. WHEN o sistema é testado com casos extremos THEN ele SHALL lidar corretamente com prefixes e suffixes de diferentes tamanhos
4. WHEN o sistema é testado com checksum THEN a validação de suffix SHALL respeitar o formato EIP-55