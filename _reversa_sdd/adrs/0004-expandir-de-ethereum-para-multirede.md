# ADR 0004 — Expandir de Ethereum para multi-rede

## Status

Aceito retroativamente.

## Contexto

O projeto começou como gerador Ethereum, mas o histórico inclui `aceb60e feat: bitcoin and solana support`. A interface de domínio ganhou `Network`, geradores específicos e diferenças por rede para private key, endereço e persistência.

## Decisão

Suportar Ethereum, Bitcoin e Solana por meio da interface `crypto.Generator`, mantendo regras específicas por rede no worker, crypto e keystore.

## Evidências

- `internal/crypto/generator.go`: contrato comum.
- `internal/crypto/ethereum.go`, `bitcoin.go`, `solana.go`: implementações concretas.
- `pkg/wallet.Wallet.Network` e `GenerationCriteria.Network`.
- Git: `aceb60e feat: bitcoin and solana support`.

## Consequências

- Amplia uso do produto além de Ethereum.
- Cria divergências de validação e persistência por rede.
- Expõe lacunas: `Wallet.IsValid()` ainda está centrado em endereço Ethereum de 40 chars; persistência Solana contém simplificações.

## Confiança

🟢 CONFIRMADO.
