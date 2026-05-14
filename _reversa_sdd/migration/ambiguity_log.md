# Ambiguity Log — Migração

## PENDENTES

Nenhum item pendente no momento.

## RESOLVIDOS COM DECISÃO HUMANA

| ID | Origem | Decisão | Impacto |
|---|---|---|---|
| AMB-PAR-001 | Paradigm Advisor | Opção 1 — Adotar paradigma natural da stack, com apetite `transformational`. | Próximos agentes podem propor reorganização interna Go mais idiomática, preservando paridade comportamental. |
| BR-HUMANA-001 | Curator | Preservar exibição de private key/mnemonic em stdout no fluxo single, com aviso de segurança. | Strategist/Designer devem manter compatibilidade de UX, mas incluir warning explícito e critérios de teste contra logging acidental. |
| BR-HUMANA-002 | Curator | Preservar escopos amplos do legado nos workflows de release por compatibilidade operacional. | Strategist deve considerar risco de supply chain e Inspector deve validar que a preservação é intencional e documentada. |
| AMB-STR-001 | Strategist | Estratégia aprovada: Big Bang controlado + Parallel Run offline obrigatório. | Designer pôde redesenhar internamente; Inspector deve bloquear release via gates de paridade offline. |
| AMB-DES-001 | Designer Fase 1 | Topologia aprovada: moderna proposta, monólito modular por capability com hexagonal leve. | Agente de codificação deve criar árvore alvo por capability, sem copiar a topologia legada 1-para-1. |
| AMB-DES-002 | Designer Fase 2 | Arquitetura, domínio e dados alvo aprovados para avançar ao Inspector. | Inspector pôde gerar specs de paridade contra a arquitetura aprovada. |

## REFERIDOS À CODIFICAÇÃO

| ID | Origem | Item | Ação esperada |
|---|---|---|---|
| COD-PAR-001 | Paradigm Advisor | Paridade comportamental deve valer mais que cópia da estrutura interna do legado. | Implementar desenho Go idiomático com testes de paridade para CLI, crypto, persistência, logging, validação por rede e performance. |
| COD-STR-001 | Strategist | Big Bang exige gates de paridade antes do release. | Implementar harness de Parallel Run offline e bloquear cutover em divergências críticas. |
| COD-DES-001 | Designer | Solana precisa de artefato seguro sem `.key` bruto. | Definir extensão/formato final, implementar round-trip e testes de não geração de `.key`. |
| COD-DES-002 | Designer | Não há banco de dados alvo; dados são artefatos locais. | Usar filesystem local com permissões, escrita segura, não sobrescrita e compatibilidade de artefatos. |
| COD-INS-001 | Inspector | Specs Gherkin são contratos, não testes executáveis. | Traduzir `parity_tests/*.feature` para testes Go/CLI/CI conforme stack alvo. |
