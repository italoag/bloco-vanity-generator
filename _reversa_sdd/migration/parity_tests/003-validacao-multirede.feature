# language: pt
# spec-id: PT-003
# rastreabilidade:
#   process_flows: _reversa_sdd/code-analysis.md § Regras de negócio embutidas; _reversa_sdd/flowcharts/internal-validation.md; _reversa_sdd/flowcharts/pkg-wallet.md
#   target_architecture: target_architecture.md BC-03 Geração Vanity; BC-04 Criptografia Multirede; AD-04
#   target_domain_model: AGG-02 VanityCriteria; AGG-04 NetworkWallet
#   paradigma_alvo: CSP/goroutines Go com procedural estruturado e interfaces leves

Funcionalidade: Validação vanity e wallet por rede
  Como operador CLI multi-rede
  Quero que critérios e carteiras sejam validados conforme a rede selecionada
  Para evitar rejeições ou aceitações incorretas herdadas da validação hex global do legado

  @paridade @critico
  Cenário: Ethereum valida prefixo/sufixo hexadecimal e checksum EIP-55
    Dado a rede ethereum
    E critérios com prefixo e sufixo hexadecimais dentro do limite de 20 caracteres
    Quando o sistema novo valida VanityCriteria com checksum habilitado
    Então os critérios são aceitos
    E o match considera maiúsculas e minúsculas conforme EIP-55
    E uma carteira Ethereum válida passa em Wallet.IsValid por rede

  @paridade @critico
  Cenário: Bitcoin aceita padrão compatível com Base58 ou bech32 quando aplicável
    Dado a rede bitcoin
    E critérios contendo caracteres válidos para o formato Bitcoin configurado
    Quando o sistema novo valida VanityCriteria
    Então os critérios são aceitos mesmo quando não são hexadecimais puros
    E uma carteira Bitcoin P2PKH com chave pública comprimida passa em Wallet.IsValid por rede

  @paridade @critico
  Cenário: Solana aceita padrão Base58 e rejeita caracteres inválidos
    Dado a rede solana
    E critérios contendo somente caracteres Base58 válidos
    Quando o sistema novo valida VanityCriteria
    Então os critérios são aceitos
    Quando os critérios contêm caractere inválido em Base58
    Então a validação falha antes da geração
    E nenhuma tentativa de worker é iniciada

  @paridade @critico
  Cenário: Regra hex global do legado não é preservada como contrato para todas as redes
    Dado critérios Bitcoin ou Solana válidos para a rede mas não hexadecimais
    Quando o sistema novo valida os critérios
    Então a validação por rede substitui a validação hex global
    E a divergência é aceita por rastrear BR-MIGRAR-009 e discard_log BR-DESCARTAR-003
