---
schemaVersion: 1
generatedAt: 2026-05-08T11:46:00Z
reversa:
  version: "1.2.34"
kind: data_migration_plan
producedBy: designer
hash: "sha256:578cc3b9a58b600448c34987ddf17b005535c9e58c588185f35d32fbf29fd22e"
---

# Data Migration Plan

> Plano de migração dos dados do legado para o sistema novo. Como não há banco, o plano cobre compatibilidade e tratamento de artefatos locais de filesystem.

## Resumo

- **Volume estimado**: desconhecido e local por instalação; Data Master confirmou 0 tabelas, 0 coleções e persistência por arquivos em `keystores/` e logs locais.
- **Janela de migração**: ver `cutover_plan.md`; o cutover é release/binário, não migração centralizada de dados.
- **Estratégia**: sem ETL centralizado; compatibilidade local + política de não sobrescrita + conversão sob demanda quando necessária.
- **Banco de dados**: não aplicável.
- **Principal mudança de dados**: substituir Solana `.key` bruto por artefato criptografado/seguro e substituir log legado inseguro por log sanitizado.

## Mapeamento legado → novo

| Origem | Destino | Tipo | Notas |
|---|---|---|---|
| ausência de banco SQL/NoSQL | ausência de banco SQL/NoSQL | preservação | Nenhuma migration, DDL, CDC, backfill ou delta. |
| `keystores/<address>.json` Ethereum KeyStore V3 | `keystores/<address>.json` KeyStore V3 alvo | compatibilidade/preservação | Manter leitura/validação quando necessário; novos arquivos seguem permissões e schema alvo. |
| `keystores/<address>.pwd` | `keystores/<address>.pwd` | preservação | Arquivo sensível; não ler/logar sem ação explícita do usuário. |
| `keystores/<address>.mnemonic` | `keystores/<address>.mnemonic` | preservação | Arquivo sensível; manter permissões e não sobrescrever silenciosamente. |
| `keystores/<address>.key` Solana bruto | `SolanaSecureArtifactFile` | substituição/conversão opcional | Não gerar novos `.key`; conversão de legado deve ser explícita e local, se implementada. |
| `wallets-YYYYMMDD.log` legado com private key | `SanitizedLogFile` | substituição | Não migrar conteúdo inseguro automaticamente; começar logs novos sanitizados. |
| release assets `bloco-vgen` | release assets `bloco-vgen` | renomeação | Distribuição muda nome do binário, mantendo compatibilidade documentada. |

## Transformações

### Transformação T-01: Preservar KeyStore V3 Ethereum

- **Aplica em**: `keystores/<address>.json`.
- **Regra**: arquivos existentes em formato KeyStore V3 devem continuar válidos; novos arquivos devem manter versão 3, AES-128-CTR, KDF permitido, salt/IV aleatórios e MAC Keccak.
- **Tratamento de inválidos**: rejeitar leitura/importação com erro estruturado; nunca regravar automaticamente.
- **Origem da regra**: BR-MIGRAR-017, BR-MIGRAR-019, BR-MIGRAR-021, Data Master.

### Transformação T-02: Preservar arquivos `.pwd` e `.mnemonic`

- **Aplica em**: `keystores/<address>.pwd`, `keystores/<address>.mnemonic`.
- **Regra**: tratar como segredo local; novos arquivos usam permissão `0600`; arquivos existentes não são exibidos/logados automaticamente.
- **Tratamento de inválidos**: warning ou erro estruturado conforme operação; não apagar sem confirmação explícita.
- **Origem da regra**: BR-MIGRAR-018, BR-MIGRAR-021, BR-MIGRAR-023.

### Transformação T-03: Substituir Solana `.key` bruto

- **Aplica em**: `keystores/<address>.key` e artefatos Solana legados.
- **Regra**: o sistema alvo não gera `.key` bruto. Se uma ferramenta futura de conversão for criada, ela deve ler o `.key` local, cifrar o material Ed25519 em `SolanaSecureArtifactFile`, validar round-trip e preservar o original sem sobrescrever até confirmação do usuário.
- **Tratamento de inválidos**: rejeitar arquivo vazio/malformado; registrar erro sanitizado; não criar artefato parcial.
- **Origem da regra**: BR-MIGRAR-016; `discard_log.md` BR-DESCARTAR-005; RISK-003.

### Transformação T-04: Encerrar log legado inseguro

- **Aplica em**: `wallets-YYYYMMDD.log`.
- **Regra**: logs legados com private key não devem ser importados para o novo log. O alvo inicia logs novos sanitizados com whitelist de campos.
- **Tratamento de inválidos**: não aplicável; arquivos antigos permanecem sob responsabilidade local do usuário.
- **Origem da regra**: BR-MIGRAR-024, BR-MIGRAR-025; `discard_log.md` BR-DESCARTAR-006; RISK-002.

### Transformação T-05: Renomear distribuição

- **Aplica em**: binários, Docker image, checksums e documentação de instalação.
- **Regra**: produto/repo/docs usam `bloco-vanity-generator`; binário compatível é `bloco-vgen`.
- **Tratamento de inválidos**: release sem binário/checksum esperado é No-Go.
- **Origem da regra**: BR-MIGRAR-002, BR-MIGRAR-030.

### Transformação T-06: Preservar stdout de segredo com warning no fluxo single

- **Aplica em**: saída textual do fluxo single, não em dados persistidos.
- **Regra**: private key/mnemonic podem ser exibidos por compatibilidade somente com aviso de segurança explícito.
- **Tratamento de inválidos**: ausência de warning em cenário que exibe segredo bloqueia release.
- **Origem da regra**: BR-HUMANA-001 resolvida.

## Estratégia de ETL

- **Ferramenta**: não há ETL obrigatório. Se conversão Solana for implementada, usar comando local explícito do próprio CLI ou utilitário de manutenção.
- **Fluxo obrigatório para release**:
  1. Não executar migração automática em diretórios do usuário.
  2. Gerar novos artefatos no formato alvo quando o usuário executar geração/persistência.
  3. Validar permissões e não vazamento nos novos artefatos.
  4. Documentar como usuários podem fazer backup manual de `keystores/` antes de atualizar.
- **Fluxo opcional futuro para Solana legado**:
  1. Usuário aponta diretório local de `keystores/`.
  2. CLI identifica `.key` Solana candidato sem logar conteúdo.
  3. CLI solicita senha ou usa política KDF configurada.
  4. CLI cria artefato seguro em arquivo novo temporário.
  5. CLI valida round-trip/importação.
  6. CLI renomeia arquivo temporário atomicamente.
  7. CLI preserva `.key` original ou move para backup somente com confirmação explícita.
- **Idempotência**: conversão opcional deve detectar destino já existente e não sobrescrever sem `--force` explícito; reexecução deve ser segura.
- **Throughput esperado**: irrelevante para banco; conversão local é O(n arquivos) e deve priorizar segurança sobre velocidade.

## Backfill e delta

- **Backfill**: não aplicável para banco. Não há dados centralizados a copiar antes do cutover.
- **Captura de delta**:
  - **Mecanismo**: não aplicável; não há escrita concorrente centralizada nem CDC.
  - **Latência aceitável**: não aplicável.
- **Reconciliação periódica**: não aplicável. Smoke tests devem validar novos artefatos gerados pelo binário alvo.

## Cutover de dados

> Ver também `cutover_plan.md`. Aqui apenas a parte específica de dados/artefatos.

- **Janela**: mesma janela do release do binário alvo.
- **Sequência de corte**:
  1. Recomendar backup manual de diretórios `keystores/` existentes antes de usar nova versão.
  2. Publicar release `bloco-vgen` com documentação de formatos persistidos.
  3. Executar smoke tests gerando Ethereum, Bitcoin e Solana em diretório temporário.
  4. Validar permissões dos arquivos gerados.
  5. Validar que Solana não gera `.key` bruto.
  6. Validar que logs novos não contêm segredo.
  7. Validar que falha de persistência é warning/status quando o resultado já foi gerado.
- **Verificação pós-corte**:
  - **Contagens**: não aplicável a tabelas; contar artefatos gerados nos smoke tests por cenário.
  - **Checksums**: checksums de release assets; não calcular checksum público de segredos de usuário.
  - **Integridade**: round-trip de KeyStore V3 e Solana secure artifact em ambiente temporário.

## Validação de qualidade

| Métrica | Alvo | Fonte de medição |
|---|---|---|
| Tabelas migradas | 0 | Data Master / ausência de DDL |
| Arquivos sensíveis novos com permissão restrita | 100% quando OS suportar | smoke tests filesystem |
| Novos `.key` Solana brutos | 0 | busca em diretório temporário de teste |
| Logs com private key/mnemonic/password/salt | 0 ocorrências | testes negativos de redaction |
| KeyStore V3 round-trip | 100% em vetores definidos | testes de crypto/keystore |
| Solana secure artifact round-trip | 100% em vetores definidos | testes de persistência Solana |
| Sobrescrita silenciosa de artefato existente | 0 ocorrências | testes de filesystem |
| Release assets com checksum | 100% | CI/release smoke test |

## Riscos específicos de dados

- **RISK-002**: vazamento de segredos em logs, stdout ou arquivos.
- **RISK-003**: Solana persistida em formato inseguro ou não recuperável.
- **RISK-013**: dados locais legados não terem migração automática.
- **RISK-010**: escopos amplos de release preservarem risco de supply chain.

## Notas

Não existe migração de banco a planejar. A migração de dados é, na prática, uma política de compatibilidade e segurança para artefatos locais. O alvo deve ser conservador com dados do usuário: não apagar, não sobrescrever e não converter segredos sem ação explícita.
