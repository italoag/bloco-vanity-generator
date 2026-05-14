# language: pt
# spec-id: PT-001
# rastreabilidade:
#   process_flows: _reversa_sdd/code-analysis.md § Fluxo principal de geração; _reversa_sdd/flowcharts/cmd-bloco-vgen.md; _reversa_sdd/flowcharts/internal-cli.md
#   target_architecture: target_architecture.md BC-01 Experiência de Comando; BC-02 Configuração Operacional; AD-01; AD-02
#   target_domain_model: AGG-01 GenerationRequest
#   paradigma_alvo: CSP/goroutines Go com procedural estruturado e interfaces leves

Funcionalidade: Inicialização CLI e configuração operacional
  Como operador CLI
  Quero iniciar o binário bloco-vgen com configuração validada e flags explícitas
  Para evitar execução com parâmetros inválidos antes de acionar geração, crypto ou filesystem

  @paridade @critico
  Cenário: Inicialização válida executa comando raiz com configuração validada
    Dado um ambiente sem variáveis inválidas
    E uma configuração padrão equivalente ao legado
    Quando o operador executa bloco-vgen com flags válidas para geração vanity
    Então o sistema novo valida a configuração antes de iniciar a geração
    E aplica somente os overrides de flags explicitamente alteradas
    E preserva o contrato externo de stdout, stderr e código de saída esperado para sucesso

  @paridade @critico
  Cenário: Configuração inválida bloqueia execução antes da geração
    Dado uma configuração com quiet e verbose habilitados simultaneamente
    Quando o operador executa bloco-vgen
    Então o sistema novo encerra com código de erro
    E não inicia worker pool
    E não gera arquivos em keystores
    E exibe erro estruturado sem stack trace quando BLOCO_DEBUG não está ativo

  @paridade @critico
  Cenário: Threads respeitam autodetecção e limites operacionais
    Dado que o operador informa --threads=0
    Quando o sistema novo monta GenerationRequest
    Então a CLI interpreta threads como autodetecção de CPU
    E o motor de geração recebe thread count operacional maior ou igual a 1
    E valores fora do limite configurável são rejeitados antes da geração

  @paridade @critico
  Cenário: Debug controlado expõe detalhes somente com BLOCO_DEBUG
    Dado uma falha operacional reproduzível na configuração
    Quando BLOCO_DEBUG está desabilitado
    Então o erro exibido não contém stack trace
    Quando BLOCO_DEBUG está habilitado
    Então o erro pode conter detalhes de diagnóstico
    E nenhum detalhe contém private key, mnemonic, password ou salt
