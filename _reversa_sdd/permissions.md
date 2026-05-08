# Permissions — Detective

> Projeto: `bloco-wallet-generator`  
> Escala: 🟢 CONFIRMADO | 🟡 INFERIDO | 🔴 LACUNA

## Conclusão rápida

O sistema é uma CLI local e não implementa RBAC/ACL de aplicação para usuários finais. As permissões encontradas são de dois tipos:

1. **Permissões operacionais locais:** capacidades de quem executa o binário sobre CPU, filesystem, stdout/stderr e arquivos sensíveis.
2. **Permissões de automação GitHub Actions:** escopos do token em CI, release, Docker e scans de segurança.

## Papéis operacionais locais

| Papel | Descrição | Confiança |
|---|---|---:|
| Operador CLI | Pessoa/processo que executa o binário localmente. | 🟢 |
| Sistema de arquivos local | Ambiente onde keystores, mnemonics e logs são gravados. | 🟢 |
| Terminal/TUI | Interface stdout/stderr usada para progresso, resultados e mensagens. | 🟢 |
| Worker goroutine | Execução concorrente interna sem identidade humana. | 🟢 |

## Matriz de permissões locais

| Ação/Recurso | Operador CLI | Worker | Logger | Observações | Confiança |
|---|---:|---:|---:|---|---:|
| Definir prefix/suffix/checksum | sim | não | não | Via flags Cobra. | 🟢 |
| Escolher rede | sim | lê | não | `--network`; default Ethereum. | 🟢 |
| Gerar chave privada | não diretamente | sim | não | Worker/crypto geram material sensível. | 🟢 |
| Exibir private key em stdout | sim, por execução normal | não | não | Modo texto mostra private key; quiet reduz exposição em múltiplos wallets. | 🟢 |
| Salvar KeyStore | sim, por config | usa resultado | não | Default habilitado salvo em diretório configurável. | 🟢 |
| Salvar mnemonic | sim, se gerada | gera/fornece | não | Bitcoin salva mnemonic; Ethereum opcional. | 🟢 |
| Registrar logs operacionais | controla por flags | emite eventos | sim | Logs devem ser sanitizados. | 🟢 |
| Registrar chaves privadas/mnemonic em logs | não permitido | não permitido | não permitido | Regra de segurança central. | 🟢 |
| Cancelar execução | sim | observa contexto | não | SIGINT/SIGTERM. | 🟢 |
| Alterar diretório de saída | sim | não | não | `--keystore-dir`, `--log-file`. | 🟢 |

## Restrições e controles locais

| Controle | Descrição | Evidência | Confiança |
|---|---|---|---:|
| File mode sensível | Config de keystore inclui modo de arquivo; intenção de proteger arquivos gerados. | `KeyStoreConfig.FileMode` | 🟡 |
| Redaction por whitelist | Logger aceita apenas campos seguros e sanitiza erros/paths. | `pkg/logging/secure_logger.go` | 🟢 |
| Salt não logado | Logger KDF remove `salt` explicitamente. | `internal/crypto/kdf/interfaces.go` | 🟢 |
| Fallback sem stdout em TUI | Logger pode desabilitar/fazer fallback para evitar interferência visual. | `shouldAvoidStdout()` | 🟢 |
| CI ignora testes longos | Testes usam `-short` e timeout para viabilidade de pipeline. | `.github/workflows/ci.yaml`, commit `4a593e0` | 🟢 |

## Permissões GitHub Actions

| Workflow | Escopos declarados | Uso provável | Confiança |
|---|---|---|---:|
| `ci.yaml` | `contents: read`, `security-events: write` | checkout e upload SARIF de gosec. | 🟢 |
| `release.yaml` | `contents: write`, `issues: write`, `packages: write`, `id-token: write`, `pull-requests: write`, `security-events: write` | criar release, publicar assets, pacotes Docker, checksums e eventos de segurança. | 🟢 |
| `docker.yaml` build | `contents: read`, `packages: write`, `security-events: write` | build/push GHCR e upload Trivy SARIF. | 🟢 |
| `docker.yaml` cleanup | `contents: read`, `packages: write` | remover versões antigas de imagens. | 🟢 |
| `semgrep.yml` | `contents: read` + `SEMGREP_APP_TOKEN` | análise Semgrep em container. | 🟢 |

## Matriz CI/CD

| Ação/Recurso | CI | Release | Docker | Semgrep | Confiança |
|---|---:|---:|---:|---:|---:|
| Ler repositório | sim | sim | sim | sim | 🟢 |
| Escrever release/assets | não | sim | não | não | 🟢 |
| Publicar pacote GHCR | não | sim | sim | não | 🟢 |
| Upload SARIF/security events | sim | sim | sim | não direto | 🟢 |
| Apagar pacotes antigos | não | não | sim, cleanup main | não | 🟢 |
| Usar token externo | não explícito | `GITHUB_TOKEN` | `GITHUB_TOKEN` | `SEMGREP_APP_TOKEN` | 🟢 |

## Lacunas de permissões

| ID | Lacuna | Impacto | Confiança |
|---|---|---|---:|
| PERM-GAP-001 | Não há RBAC/ACL de aplicação porque é CLI local. | Não aplicável para controle por usuário dentro do app. | 🟢 |
| PERM-GAP-002 | O modo texto pode exibir private key/mnemonic no terminal, dependendo flags e fluxo. | Risco operacional se stdout for capturado. | 🟢 |
| PERM-GAP-003 | Logs legados `wallets-*.log` existem em `internal/worker` e parecem conter termo de private key. | Risco histórico de segredo em repositório/ambiente. | 🟢 |
| PERM-GAP-004 | Release workflow pede escopos amplos (`issues`, `pull-requests`, `id-token`) cuja necessidade não foi provada nos passos lidos. | Princípio do menor privilégio pode estar relaxado. | 🟡 |
