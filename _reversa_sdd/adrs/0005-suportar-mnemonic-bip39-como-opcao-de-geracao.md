# ADR 0005 — Suportar mnemonic BIP-39 como opção de geração

## Status

Aceito retroativamente.

## Contexto

O histórico registra `b1e416c Add mnemonic-based vanity wallet generation` e `6c80aa2 Save mnemonic phrases alongside keystores`. O README lista vanity generation using BIP-39 mnemonic phrases or raw private keys.

## Decisão

Permitir geração vanity usando mnemonic quando solicitado por `--with-mnemonic`, além do fluxo padrão com private key aleatória. Salvar mnemonic junto aos artefatos quando disponível.

## Evidências

- README: `--with-mnemonic` e feature BIP-39.
- `pkg/wallet.Wallet.Mnemonic` e `GenerationCriteria.UseMnemonic`.
- `internal/worker/pool.go`: lógica para Ethereum mnemonic.
- `internal/cli/commands.go`: salvamento de mnemonic.
- Git: `b1e416c`, `6c80aa2`.

## Consequências

- Melhora recoverability/backups para usuários.
- Aumenta criticidade de higiene de segredos: mnemonic é segredo equivalente à chave.
- Suporte é desigual entre redes; no worker, mnemonic é efetivo apenas para Ethereum.

## Confiança

🟢 CONFIRMADO.
