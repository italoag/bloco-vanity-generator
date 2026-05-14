# language: pt
# spec-id: PT-005
# rastreabilidade:
#   process_flows: _reversa_sdd/code-analysis.md § Fluxo principal de geração; _reversa_sdd/database/data-dictionary.md; _reversa_sdd/database/relationships.md
#   target_architecture: target_architecture.md BC-05 Cofre Local e Artefatos; AD-05; AD-07
#   target_domain_model: AGG-05 SecureArtifactSet
#   target_data_model: target_data_model.md § Entidades de dados; § Restrições
#   paradigma_alvo: CSP/goroutines Go com procedural estruturado e interfaces leves

Funcionalidade: Persistência segura de artefatos locais
  Como operador que habilita persistência local
  Quero que keystore, senha, mnemonic e artefatos Solana sejam gravados com segurança
  Para preservar backup sem expor material sensível ou sobrescrever dados existentes

  @paridade @critico
  Cenário: Ethereum persiste KeyStore, senha e mnemonic com permissões restritas
    Dado uma carteira Ethereum gerada com keystore habilitado
    E um diretório temporário de keystores vazio
    Quando o sistema novo persiste os artefatos
    Então cria arquivo KeyStore V3 JSON quando aplicável
    E cria arquivo .pwd com a senha quando aplicável
    E cria arquivo .mnemonic quando mnemonic está presente
    E cada arquivo sensível tem permissão restrita quando o sistema operacional suportar

  @paridade @critico
  Cenário: Bitcoin persiste backup por mnemonic sem KeyStore V3
    Dado uma carteira Bitcoin com mnemonic presente
    Quando o sistema novo persiste o backup
    Então cria arquivo .mnemonic seguro
    E não cria KeyStore V3 Ethereum para Bitcoin
    E uma carteira Bitcoin sem mnemonic gera erro de backup recuperável conforme política alvo

  @paridade @critico
  Cenário: Solana não gera arquivo .key bruto
    Dado uma carteira Solana válida
    E persistência local habilitada
    Quando o sistema novo persiste artefatos Solana
    Então nenhum arquivo .key bruto é criado
    E um artefato Solana seguro criptografado é criado ou a funcionalidade é bloqueada como experimental de forma explícita
    E o artefato seguro passa em round-trip de recuperação em ambiente de teste

  @paridade @critico
  Cenário: Falha de persistência vira warning sem apagar resultado gerado
    Dado uma carteira válida já gerada
    E um diretório de saída sem permissão de escrita
    Quando o sistema novo tenta persistir os artefatos durante a exibição
    Então a carteira ainda é exibida conforme política de saída
    E a falha de persistência é comunicada como warning ou status
    E nenhum arquivo parcial sensível fica com permissão insegura

  @paridade @critico
  Cenário: Artefato existente não é sobrescrito silenciosamente
    Dado um arquivo sensível existente para o mesmo endereço no diretório de saída
    Quando o sistema novo tenta persistir novo artefato no mesmo caminho
    Então a operação não sobrescreve silenciosamente o arquivo existente
    E o usuário recebe erro ou warning estruturado
    E o comportamento é idempotente em reexecuções sem --force explícito
