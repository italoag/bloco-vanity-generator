# language: pt
# spec-id: PT-002
# rastreabilidade:
#   process_flows: _reversa_sdd/code-analysis.md § Loop concorrente de geração; _reversa_sdd/flowcharts/internal-cli.md; _reversa_sdd/flowcharts/internal-worker.md
#   target_architecture: target_architecture.md BC-03 Geração Vanity; AD-03; Bordas com o legado durante a migração
#   target_domain_model: AGG-03 WalletGeneration
#   paradigma_alvo: CSP/goroutines Go com procedural estruturado e interfaces leves

Funcionalidade: Geração vanity com worker pool concorrente
  Como operador CLI
  Quero gerar carteiras vanity usando concorrência controlada
  Para manter desempenho, cancelamento gracioso e resultados observáveis equivalentes ao legado

  @paridade @critico
  Cenário: Geração single retorna primeiro resultado vencedor válido
    Dado critérios válidos para uma rede suportada
    E thread count configurado para múltiplos workers
    Quando o operador executa geração single
    Então o sistema novo inicia workers concorrentes com contexto cancelável
    E retorna uma carteira cujo endereço satisfaz prefixo e sufixo configurados
    E reporta tentativas, duração e worker vencedor
    E encerra os workers restantes sem race ou goroutine vazada observável

  @paridade @critico
  Cenário: Geração múltipla preserva sucesso parcial quando um item falha
    Dado critérios válidos para geração múltipla
    E uma falha injetada em um item intermediário
    Quando o operador executa count maior que 1 em modo texto
    Então o sistema novo registra erro para o item falho
    E continua tentando gerar os itens restantes
    E apresenta resumo com sucessos e falhas sem abortar todo o lote

  @paridade @critico
  Cenário: Cancelamento por sinal encerra geração longa de forma graciosa
    Dado uma geração probabilisticamente longa em execução
    Quando o processo recebe interrupção equivalente a SIGINT ou SIGTERM
    Então o contexto de geração é cancelado
    E workers param de tentar novas chaves
    E a CLI retorna controle sem corromper arquivos parcialmente escritos
    E o erro ou status de cancelamento é observável em stderr ou saída apropriada

  @paridade @critico
  Cenário: Estatísticas de progresso são monotônicas e livres de race
    Dado uma geração com progresso habilitado
    Quando workers enviam estatísticas periódicas
    Então tentativas agregadas nunca diminuem
    E velocidade e ETA são atualizados sem data race detectável
    E nenhuma atualização de progresso contém private key, mnemonic, password ou salt
