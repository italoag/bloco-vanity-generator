# language: pt
# spec-id: PT-004
# rastreabilidade:
#   process_flows: _reversa_sdd/code-analysis.md § Geração de endereço Ethereum; § Checksum EIP-55; § KeyStore V3; § KDF universal; _reversa_sdd/flowcharts/internal-crypto.md; _reversa_sdd/flowcharts/internal-crypto-kdf.md
#   target_architecture: target_architecture.md BC-04 Criptografia Multirede; BC-05 Cofre Local e Artefatos; AD-05
#   target_domain_model: AGG-04 NetworkWallet; AGG-05 SecureArtifactSet
#   paradigma_alvo: CSP/goroutines Go com procedural estruturado e interfaces leves

Funcionalidade: Criptografia Ethereum, KeyStore V3 e KDF
  Como operador que salva carteira Ethereum
  Quero que geração, KeyStore e KDF preservem compatibilidade e segurança
  Para recuperar a carteira e interoperar com clientes Ethereum sem expor segredos

  @paridade @critico
  Cenário: Ethereum deriva endereço por secp256k1, Keccak e EIP-55
    Dado uma private key determinística de 32 bytes para teste
    Quando o sistema novo deriva a carteira Ethereum
    Então o endereço gerado corresponde ao vetor de referência por secp256k1 e Keccak
    E o checksum EIP-55 corresponde ao vetor de referência
    E a private key não aparece em logs ou mensagens de erro

  @paridade @critico
  Cenário: KeyStore V3 preserva formato Ethereum esperado
    Dado uma carteira Ethereum válida
    E uma senha segura gerada pelo sistema
    Quando o sistema novo gera KeyStore V3
    Então o JSON contém version igual a 3
    E cipher igual a aes-128-ctr
    E kdf permitido entre scrypt, pbkdf2, pbkdf2-sha256 ou pbkdf2-sha512
    E mac calculado sobre derivedKey[16:32] mais ciphertext
    E o KeyStore passa em round-trip de descriptografia com a senha correta

  @paridade @critico
  Cenário: Parâmetros KDF inválidos bloqueiam derivação e não vazam salt
    Dado parâmetros KDF inválidos para scrypt ou PBKDF2
    Quando o sistema novo tenta derivar a chave
    Então a operação falha com erro estruturado de KDF
    E nenhum KeyStore é persistido
    E logs e stderr não contêm salt, password, derived key, ciphertext, IV ou MAC

  @paridade @critico
  Cenário: Senha gerada satisfaz política mínima de segurança
    Dado uma solicitação de persistência Ethereum com senha gerada automaticamente
    Quando o sistema novo gera a senha
    Então a senha tem no mínimo 12 caracteres
    E contém pelo menos uma letra minúscula, uma maiúscula, um número e um caractere especial
    E a senha é persistida apenas em artefato sensível com permissão restrita quando habilitado
