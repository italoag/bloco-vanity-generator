# language: pt
# spec-id: PT-007
# rastreabilidade:
#   process_flows: _reversa_sdd/code-analysis.md § Concorrência e estado; _reversa_sdd/flowcharts/internal-cli.md; _reversa_sdd/flowcharts/internal-progress.md; _reversa_sdd/flowcharts/internal-tui.md
#   target_architecture: target_architecture.md BC-01 Experiência de Comando; BC-03 Geração Vanity; Adapter terminal
#   target_domain_model: AGG-06 TerminalSession
#   paradigma_alvo: CSP/goroutines Go com procedural estruturado e interfaces leves

Funcionalidade: TUI, fallback textual e progress manager sem deadlock
  Como operador em diferentes terminais
  Quero progresso robusto em TUI ou texto
  Para acompanhar geração sem travamentos e sem quebrar automação

  @paridade @critico
  Cenário: TUI é usada somente quando ambiente e flags permitem
    Dado TUI habilitada
    E progresso habilitado
    E quiet desabilitado
    E terminal compatível
    Quando o operador inicia uma geração
    Então o sistema novo usa TUI para progresso e estatísticas
    E a lógica de geração permanece desacoplada da renderização terminal

  @paridade @critico
  Cenário: Ambiente sem suporte visual usa fallback texto
    Dado TERM vazio ou igual a dumb
    E NO_COLOR habilitado ou equivalente
    E stdout redirecionado ou execução em CI
    Quando o operador inicia geração com progresso
    Então o sistema novo evita TUI quando apropriado
    E usa fallback texto ou saída simples
    E mantém códigos de saída e resultados equivalentes ao fluxo sem TUI

  @paridade @critico
  Cenário: Falha de TUI cai para texto sem abortar geração
    Dado uma falha injetada ao iniciar Bubble Tea ou terminal UI
    Quando o sistema novo tenta renderizar a TUI
    Então o erro de TUI é tratado como fallback
    E a geração continua em modo texto quando seguro
    E o usuário recebe aviso ou status sem stack trace em modo normal

  @paridade @critico
  Cenário: Progress manager textual inicia e para sem deadlock
    Dado progress manager textual habilitado como fallback
    Quando uma geração longa inicia e depois é concluída ou cancelada
    Então Start e Stop são idempotentes
    E não há deadlock ou goroutine pendurada observável
    E o teste com timeout finaliza dentro da janela definida

  @paridade @critico
  Cenário: Progresso textual não vaza segredos
    Dado uma geração com progress textual ativo
    Quando o sistema novo atualiza tentativas, velocidade, ETA e status
    Então a saída de progresso não contém private key, mnemonic, password, salt ou material criptográfico
