# ADR 0002 — Usar KeyStore V3 com KDF universal

## Status

Aceito retroativamente.

## Contexto

O produto precisa gerar carteiras importáveis por clientes Ethereum e armazenar chaves privadas com proteção criptográfica. O histórico mostra introdução de KeyStore V3 (`c526b25`, `5c3fe85`, `6c4cdd4`) e evolução para KDF universal com scrypt/PBKDF2 e análise de compatibilidade.

## Decisão

Persistir chaves Ethereum/Solana por fluxo de keystore, usando formato KeyStore V3 quando aplicável, AES-128-CTR, MAC Keccak e KDF configurável (`scrypt`, `pbkdf2`, `pbkdf2-sha256`, `pbkdf2-sha512`).

## Evidências

- README: KeyStore V3 Generation with Universal KDF.
- `internal/crypto/keystore.go`: `KeyStoreV3`, `GenerateMAC`, AES-128-CTR, validação de versão/cipher/KDF.
- `internal/crypto/kdf`: serviço universal, handlers e analyzer.
- Git: commits `c526b25`, `a5a0a0a`.

## Consequências

- Aumenta compatibilidade com MetaMask/geth/Besu/Anvil/Reth/Firefly.
- Introduz complexidade de validação e performance de KDF.
- Parâmetros fracos/incompatíveis precisam ser detectados e comunicados.

## Confiança

🟢 CONFIRMADO.
