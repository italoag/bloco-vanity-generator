# Auditoria de segurança — bloco-vanity-generator

Relatório da auditoria manual de segurança do repositório.

## Arquivos

| Arquivo | Descrição |
|---|---|
| `relatorio-auditoria-seguranca.pdf` | Relatório completo em PDF (A4, 29 páginas), com capa, gráficos, tabelas de achados e as issues prontas para o GitHub |
| `relatorio-auditoria-seguranca.md` | Mesmo conteúdo em Markdown, para consumo por agentes de correção |
| `achados.py` | Base de dados da auditoria: achados, pontos fortes, recomendações e issues. **Edite apenas este arquivo para atualizar o relatório** |
| `gerar_relatorio.py` | Gerador: formata `achados.py` em PDF e Markdown e desenha os gráficos |
| `graficos/` | PNGs gerados (rosca por severidade, barras por categoria) |

## Regerar o relatório

O ambiente é isolado num venv local; nada é instalado globalmente.

```bash
cd docs/security-audit

# primeira vez
python3 -m venv .venv
.venv/bin/pip install reportlab matplotlib

# a cada regeração
.venv/bin/python gerar_relatorio.py
```

O script reescreve `relatorio-auditoria-seguranca.pdf`, `relatorio-auditoria-seguranca.md`
e os PNGs em `graficos/`.

## Verificação visual (opcional)

```bash
.venv/bin/pip install pymupdf
.venv/bin/python - <<'PY'
import pymupdf
d = pymupdf.open("relatorio-auditoria-seguranca.pdf")
print("páginas:", d.page_count)
for i, p in enumerate(d):
    p.get_pixmap(dpi=100).save("/tmp/p%02d.png" % (i + 1))
    for b in p.get_text("blocks"):
        x0, _, x1, _ = b[:4]
        if x1 > 541 or x0 < 53:      # margens A4 de 2 cm
            print("página %d fora da margem: %r" % (i + 1, b[4][:60]))
PY
```

## Paleta de severidades

| Severidade | Cor |
|---|---|
| Crítica | `#B91C1C` |
| Alta | `#EA580C` |
| Média | `#D97706` |
| Baixa | `#2563EB` |
| Ponto forte | `#059669` |
