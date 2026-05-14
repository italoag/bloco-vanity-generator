# language: pt
# spec-id: PT-008
# rastreabilidade:
#   process_flows: _reversa_sdd/inventory.md § CI/CD e Docker; _reversa_sdd/domain.md § CI/CD e governança técnica; _reversa_sdd/migration/ambiguity_log.md
#   target_architecture: target_architecture.md BC-07 Distribuição e Qualidade; AD-06; target_architecture.md § Bordas com o legado durante a migração
#   target_domain_model: AGG-08 ReleaseContract
#   paradigma_alvo: CSP/goroutines Go com procedural estruturado e interfaces leves

Funcionalidade: CI/CD, Docker, release e permissões operacionais
  Como mantenedor do projeto
  Quero releases verificáveis e compatíveis com a operação legada
  Para distribuir bloco-vgen com qualidade, checksums e scans de segurança

  @paridade @critico
  Cenário: CI executa qualidade mínima antes do release
    Dado um release candidate do sistema novo
    Quando o pipeline de CI é executado
    Então testes Go são executados
    E race detector, go vet, gofmt, lint, gosec ou equivalente e govulncheck ou equivalente são executados conforme disponibilidade
    E falha em etapa crítica bloqueia release

  @paridade @critico
  Cenário: Release publica binários e checksums esperados
    Dado uma tag de release válida
    Quando o workflow de release publica artefatos
    Então existem binários para Linux e macOS em amd64 e arm64 quando suportado
    E existe checksum verificável para cada binário
    E o binário canônico é bloco-vgen
    E a documentação não trata bloco-eth como nome canônico alvo

  @paridade @critico
  Cenário: Docker usa multi-stage e runtime não-root
    Dado a imagem Docker alvo
    Quando a imagem é construída
    Então o build usa estágio Go e runtime mínimo
    E o processo no runtime não executa como root
    E o comando de ajuda ou versão do bloco-vgen funciona no container

  @paridade @critico
  Cenário: Escopos amplos de release são preservados com rastreabilidade
    Dado a decisão humana de preservar escopos amplos por compatibilidade operacional
    Quando os workflows alvo são revisados
    Então os escopos preservados estão documentados com justificativa
    E branch protection e revisão de workflow são recomendadas como mitigação
    E a divergência em relação ao menor privilégio é rastreada como risco aceito

  @paridade @critico
  Cenário: Parallel Run offline bloqueia release com divergência crítica
    Dado um release candidate do bloco-vgen
    Quando a suíte de paridade offline compara legado e alvo
    Então divergências intencionais são marcadas com referência ao discard_log ou target_business_rules
    E qualquer divergência crítica não explicada bloqueia publicação
