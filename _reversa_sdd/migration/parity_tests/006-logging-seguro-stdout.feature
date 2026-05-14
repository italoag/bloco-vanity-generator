# language: pt
# spec-id: PT-006
# rastreabilidade:
#   process_flows: _reversa_sdd/code-analysis.md § Logging; _reversa_sdd/flowcharts/pkg-logging.md; _reversa_sdd/migration/ambiguity_log.md
#   target_architecture: target_architecture.md BC-01 Experiência de Comando; BC-06 Observabilidade Segura; AD-05
#   target_domain_model: AGG-06 TerminalSession; AGG-07 SecureTelemetry
#   paradigma_alvo: CSP/goroutines Go com procedural estruturado e interfaces leves

Funcionalidade: Logging seguro e exibição controlada de segredos em stdout
  Como operador CLI
  Quero logs sanitizados e avisos claros quando segredos forem exibidos
  Para preservar compatibilidade sem vazar material sensível em canais indevidos

  @paridade @critico
  Cenário: Logs de carteira gerada usam whitelist e não contêm segredos
    Dado uma geração bem-sucedida com logging habilitado
    Quando o sistema novo registra wallet_generated
    Então o log pode conter endereço, rede, tentativas, duração, worker e status
    E o log não contém private key, public key, mnemonic, password, salt, ciphertext, IV, MAC ou material KDF
    E qualquer campo não permitido é removido ou mascarado antes da escrita

  @paridade @critico
  Cenário: Salt e parâmetros sensíveis de KDF nunca são logados
    Dado uma operação KDF com análise ou verbose habilitado
    Quando o sistema novo registra tentativa, sucesso ou erro de KDF
    Então o relatório pode conter algoritmo, compatibilidade e nível de segurança
    E não contém salt, password, derived key, ciphertext, IV ou MAC

  @paridade @critico
  Cenário: Fluxo single pode exibir segredo somente com warning de segurança
    Dado uma geração single em modo texto
    E a decisão humana de preservar exibição de private key ou mnemonic por compatibilidade
    Quando o sistema novo exibe private key ou mnemonic em stdout
    Então um aviso de segurança explícito é exibido no mesmo fluxo
    E o aviso informa risco de captura de stdout ou histórico de terminal
    E logs continuam sem conter o segredo exibido

  @paridade @critico
  Cenário: Quiet reduz exposição operacional
    Dado uma geração múltipla com quiet habilitado
    Quando o sistema novo completa uma ou mais carteiras
    Então private key e mnemonic não são exibidos por carteira
    E logs permanecem sanitizados
    E o código de saída reflete sucesso ou falha conforme contrato CLI

  @paridade @critico
  Cenário: Logging desabilitado descarta eventos sem interferir na geração
    Dado logging desabilitado por flag ou configuração
    Quando uma carteira é gerada com sucesso
    Então nenhum arquivo de log operacional é criado pelo logger seguro
    E a geração e persistência configurada continuam funcionando
