# ADR 0001 — Adotar logging seguro e sanitizado

## Status

Aceito retroativamente.

## Contexto

O README contém um aviso explícito: versões anteriores podem ter registrado private keys e outros dados sensíveis. O commit `ece8dcf` (`feat: secure logging`) adicionou um subsistema completo em `pkg/logging` com testes de sanitização, rotação, formatos e integração com workers.

## Decisão

Registrar somente dados operacionais não sensíveis: endereço, tentativas, duração, thread, status, contadores e métricas. Private keys, public keys, mnemonics, salts e material criptográfico não devem ser registrados.

## Evidências

- README: Security Notice e Secure Logging System.
- `pkg/logging/secure_logger.go`: whitelist de parâmetros seguros e sanitização.
- `internal/crypto/kdf/interfaces.go`: `salt` explicitamente removido do logging.
- Git: `ece8dcf feat: secure logging`.

## Consequências

- Melhora postura de segurança e reduz risco de vazamento por logs.
- Logs antigos devem ser tratados como potencialmente sensíveis.
- Debugging precisa operar com contexto sanitizado, não com segredo bruto.

## Confiança

🟢 CONFIRMADO.
