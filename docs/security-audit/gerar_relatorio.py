# -*- coding: utf-8 -*-
"""
Gerador do relatório de auditoria de segurança do bloco-vanity-generator.

Produz:
  - relatório-auditoria-segurança.pdf  (A4, margens 2cm, cabeçalho e rodapé)
  - relatório-auditoria-segurança.md   (mesmo conteúdo, para agentes de correção)
  - gráficos/*.png                     (rosca por severidade, barras por categoria)

Uso (a partir de docs/security-audit/):
    .venv/bin/python gerar_relatorio.py
"""

import os
import sys
from collections import Counter, OrderedDict

BASE = os.path.dirname(os.path.abspath(__file__))
sys.path.insert(0, BASE)

import matplotlib

matplotlib.use("Agg")
import matplotlib.pyplot as plt

from reportlab.lib import colors
from reportlab.lib.enums import TA_CENTER, TA_JUSTIFY, TA_LEFT
from reportlab.lib.pagesizes import A4
from reportlab.lib.styles import ParagraphStyle, getSampleStyleSheet
from reportlab.lib.units import cm
from reportlab.platypus import (
    BaseDocTemplate,
    Frame,
    Image,
    KeepTogether,
    NextPageTemplate,
    PageBreak,
    PageTemplate,
    Paragraph,
    Spacer,
    Table,
    TableStyle,
)

import achados as A

PDF_PATH = os.path.join(BASE, "relatorio-auditoria-seguranca.pdf")
MD_PATH = os.path.join(BASE, "relatorio-auditoria-seguranca.md")
GRAF_DIR = os.path.join(BASE, "graficos")
TITULO_RELATORIO = "Relatório de Auditoria de Segurança - %s" % A.PROJETO

ORDEM_SEV = ["critica", "alta", "media", "baixa", "informativa"]
C = {k: colors.HexColor(v) for k, v in A.CORES.items()}
TINTA = colors.HexColor("#0F172A")
TINTA_SUAVE = colors.HexColor("#475569")
LINHA = colors.HexColor("#CBD5E1")
FUNDO_SUAVE = colors.HexColor("#F1F5F9")


# --------------------------------------------------------------------------
# Utilitarios
# --------------------------------------------------------------------------


def esc(texto):
    """Escapa texto para os mini-tags do Paragraph do reportlab."""
    return texto.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")


def contar_por_severidade():
    c = Counter(a["severidade"] for a in A.ACHADOS)
    return OrderedDict((s, c.get(s, 0)) for s in ORDEM_SEV)


def contar_por_categoria():
    c = Counter(a["categoria"] for a in A.ACHADOS)
    # ordena por quantidade desc, depois alfabetico, para um grafico estavel
    return OrderedDict(sorted(c.items(), key=lambda kv: (-kv[1], kv[0])))


def pior_severidade(cat):
    sevs = [a["severidade"] for a in A.ACHADOS if a["categoria"] == cat]
    for s in ORDEM_SEV:
        if s in sevs:
            return s
    return "informativa"

    # --------------------------------------------------------------------------
    # Graficos
    # --------------------------------------------------------------------------


def gerar_graficos():
    os.makedirs(GRAF_DIR, exist_ok=True)
    plt.rcParams["font.family"] = "DejaVu Sans"

    # --- Rosca por severidade -------------------------------------------
    por_sev = contar_por_severidade()
    labels, valores, cores = [], [], []
    for sev, qtd in por_sev.items():
        if qtd:
            labels.append("%s\n(%d)" % (A.ROTULO_SEVERIDADE[sev].title(), qtd))
            valores.append(qtd)
            cores.append(A.CORES[sev])

    fig, ax = plt.subplots(figsize=(5.6, 4.0), dpi=200)
    wedges, textos, autotextos = ax.pie(
        valores,
        labels=labels,
        colors=cores,
        autopct="%1.0f%%",
        startangle=90,
        counterclock=False,
        pctdistance=0.76,
        wedgeprops=dict(width=0.42, edgecolor="white", linewidth=2.2),
        textprops=dict(fontsize=9.5, color="#0F172A"),
    )
    for t in autotextos:
        t.set_color("white")
        t.set_fontweight("bold")
        t.set_fontsize(9)
    ax.text(
        0,
        0.10,
        str(len(A.ACHADOS)),
        ha="center",
        va="center",
        fontsize=27,
        fontweight="bold",
        color="#0F172A",
    )
    ax.text(0, -0.19, "achados", ha="center", va="center", fontsize=10, color="#475569")
    ax.set(aspect="equal")
    fig.tight_layout()
    caminho_rosca = os.path.join(GRAF_DIR, "severidade-rosca.png")
    fig.savefig(caminho_rosca, transparent=True, bbox_inches="tight")
    plt.close(fig)

    # --- Barras por categoria -------------------------------------------
    por_cat = contar_por_categoria()
    nomes = list(por_cat.keys())
    qtds = [por_cat[n] for n in nomes]
    cores_barra = [A.CORES[pior_severidade(n)] for n in nomes]

    # quebra rotulos longos em duas linhas
    def quebrar(s, largura=30):
        palavras, linhas, atual = s.split(), [], ""
        for p in palavras:
            if len(atual) + len(p) + 1 <= largura:
                atual = (atual + " " + p).strip()
            else:
                linhas.append(atual)
                atual = p
        if atual:
            linhas.append(atual)
        return "\n".join(linhas)

    rotulos = [quebrar(n) for n in nomes]

    fig, ax = plt.subplots(figsize=(8.4, 4.3), dpi=200)
    y = range(len(nomes))
    barras = ax.barh(list(y), qtds, color=cores_barra, height=0.62)
    ax.set_yticks(list(y))
    ax.set_yticklabels(rotulos, fontsize=9, color="#0F172A")
    ax.invert_yaxis()
    ax.set_xlabel("Número de achados", fontsize=9.5, color="#475569")
    ax.set_xlim(0, max(qtds) + 1)
    ax.set_xticks(range(0, max(qtds) + 2))
    ax.tick_params(axis="x", labelsize=9, colors="#475569")
    for s in ("top", "right", "left"):
        ax.spines[s].set_visible(False)
    ax.spines["bottom"].set_color("#CBD5E1")
    ax.grid(axis="x", color="#E2E8F0", linewidth=0.8)
    ax.set_axisbelow(True)
    for barra, q in zip(barras, qtds):
        ax.text(
            barra.get_width() + 0.09,
            barra.get_y() + barra.get_height() / 2,
            str(q),
            va="center",
            fontsize=9.5,
            fontweight="bold",
            color="#0F172A",
        )
    fig.tight_layout()
    caminho_barras = os.path.join(GRAF_DIR, "categoria-barras.png")
    fig.savefig(caminho_barras, transparent=True, bbox_inches="tight")
    plt.close(fig)

    return caminho_rosca, caminho_barras

    # --------------------------------------------------------------------------
    # Estilos do PDF
    # --------------------------------------------------------------------------


def construir_estilos():
    ss = getSampleStyleSheet()
    e = {}
    e["capa_titulo"] = ParagraphStyle(
        "capa_titulo",
        parent=ss["Title"],
        fontName="Helvetica-Bold",
        fontSize=25,
        leading=31,
        textColor=TINTA,
        alignment=TA_LEFT,
        spaceAfter=4,
    )
    e["capa_sub"] = ParagraphStyle(
        "capa_sub",
        parent=ss["Normal"],
        fontName="Helvetica",
        fontSize=13,
        leading=18,
        textColor=TINTA_SUAVE,
        alignment=TA_LEFT,
    )
    e["h1"] = ParagraphStyle(
        "h1",
        parent=ss["Heading1"],
        fontName="Helvetica-Bold",
        fontSize=16.5,
        leading=21,
        textColor=TINTA,
        spaceBefore=16,
        spaceAfter=9,
    )
    e["h2"] = ParagraphStyle(
        "h2",
        parent=ss["Heading2"],
        fontName="Helvetica-Bold",
        fontSize=12.5,
        leading=16,
        textColor=TINTA,
        spaceBefore=12,
        spaceAfter=5,
    )
    e["h3"] = ParagraphStyle(
        "h3",
        parent=ss["Heading3"],
        fontName="Helvetica-Bold",
        fontSize=10.5,
        leading=14,
        textColor=TINTA_SUAVE,
        spaceBefore=11,
        spaceAfter=6,
    )
    e["corpo"] = ParagraphStyle(
        "corpo",
        parent=ss["Normal"],
        fontName="Helvetica",
        fontSize=9.6,
        leading=14.2,
        textColor=TINTA,
        alignment=TA_JUSTIFY,
        spaceAfter=6,
    )
    e["corpo_esq"] = ParagraphStyle("corpo_esq", parent=e["corpo"], alignment=TA_LEFT)
    e["celula"] = ParagraphStyle(
        "celula",
        parent=ss["Normal"],
        fontName="Helvetica",
        fontSize=8.4,
        leading=11.6,
        textColor=TINTA,
        alignment=TA_LEFT,
    )
    e["celula_cab"] = ParagraphStyle(
        "celula_cab",
        parent=ss["Normal"],
        fontName="Helvetica-Bold",
        fontSize=8.4,
        leading=11.6,
        textColor=colors.white,
        alignment=TA_LEFT,
    )
    e["celula_mono"] = ParagraphStyle(
        "celula_mono",
        parent=ss["Normal"],
        fontName="Courier",
        fontSize=7.5,
        leading=10.2,
        textColor=TINTA,
    )
    e["codigo"] = ParagraphStyle(
        "codigo",
        parent=ss["Normal"],
        fontName="Courier",
        fontSize=7.4,
        leading=9.8,
        textColor=colors.HexColor("#1E293B"),
        backColor=colors.HexColor("#F8FAFC"),
        borderColor=LINHA,
        borderWidth=0.6,
        borderPadding=6,
        leftIndent=1,
        rightIndent=1,
        spaceBefore=10,
        spaceAfter=12,
    )
    e["nota"] = ParagraphStyle(
        "nota",
        parent=ss["Normal"],
        fontName="Helvetica-Oblique",
        fontSize=8.7,
        leading=12.4,
        textColor=TINTA_SUAVE,
        alignment=TA_JUSTIFY,
        spaceAfter=5,
    )
    e["chip"] = ParagraphStyle(
        "chip",
        parent=ss["Normal"],
        fontName="Helvetica-Bold",
        fontSize=7.6,
        leading=10,
        textColor=colors.white,
        alignment=TA_CENTER,
    )
    e["issue_cab"] = ParagraphStyle(
        "issue_cab",
        parent=ss["Normal"],
        fontName="Courier-Bold",
        fontSize=8.6,
        leading=12,
        textColor=colors.HexColor("#B91C1C"),
        spaceBefore=10,
        spaceAfter=4,
    )
    e["issue_md"] = ParagraphStyle(
        "issue_md",
        parent=ss["Normal"],
        fontName="Courier",
        fontSize=7.3,
        leading=10.4,
        textColor=colors.HexColor("#0F172A"),
        alignment=TA_LEFT,
        backColor=colors.HexColor("#F8FAFC"),
        borderPadding=(0, 6, 0, 6),
        leftIndent=0,
        rightIndent=0,
        spaceBefore=0,
        spaceAfter=0,
        allowWidows=1,
        allowOrphans=1,
    )
    e["kpi_num"] = ParagraphStyle(
        "kpi_num",
        parent=ss["Normal"],
        fontName="Helvetica-Bold",
        fontSize=21,
        leading=24,
        alignment=TA_CENTER,
        textColor=colors.white,
    )
    e["kpi_lbl"] = ParagraphStyle(
        "kpi_lbl",
        parent=ss["Normal"],
        fontName="Helvetica-Bold",
        fontSize=7.6,
        leading=10,
        alignment=TA_CENTER,
        textColor=colors.white,
    )
    return e


def chip_severidade(sev, estilos):
    """Chip colorido de severidade, usado nas tabelas de achados."""
    t = Table(
        [[Paragraph(A.ROTULO_SEVERIDADE[sev], estilos["chip"])]],
        colWidths=[2.05 * cm],
        rowHeights=[0.52 * cm],
    )
    t.setStyle(
        TableStyle(
            [
                ("BACKGROUND", (0, 0), (-1, -1), C[sev]),
                ("VALIGN", (0, 0), (-1, -1), "MIDDLE"),
                ("LEFTPADDING", (0, 0), (-1, -1), 1),
                ("RIGHTPADDING", (0, 0), (-1, -1), 1),
                ("TOPPADDING", (0, 0), (-1, -1), 1),
                ("BOTTOMPADDING", (0, 0), (-1, -1), 1),
            ]
        )
    )
    return t

    # --------------------------------------------------------------------------
    # Cabecalho / rodape
    # --------------------------------------------------------------------------


def desenhar_moldura(canvas, doc):
    canvas.saveState()
    largura, altura = A4
    # cabecalho
    canvas.setFont("Helvetica", 7.6)
    canvas.setFillColor(TINTA_SUAVE)
    canvas.drawString(2 * cm, altura - 1.28 * cm, TITULO_RELATORIO)
    canvas.drawRightString(largura - 2 * cm, altura - 1.28 * cm, A.DATA_AUDITORIA)
    canvas.setStrokeColor(LINHA)
    canvas.setLineWidth(0.6)
    canvas.line(2 * cm, altura - 1.48 * cm, largura - 2 * cm, altura - 1.48 * cm)
    # rodape
    canvas.line(2 * cm, 1.42 * cm, largura - 2 * cm, 1.42 * cm)
    canvas.setFont("Helvetica", 7.6)
    canvas.drawString(2 * cm, 1.02 * cm, "%s - commit %s" % (A.PROJETO, A.COMMIT))
    canvas.drawRightString(largura - 2 * cm, 1.02 * cm, "Página %d" % doc.page)
    canvas.restoreState()


def desenhar_capa(canvas, doc):
    canvas.saveState()
    largura, altura = A4
    canvas.setFillColor(colors.HexColor("#0F172A"))
    canvas.rect(0, altura - 6.3 * cm, largura, 6.3 * cm, stroke=0, fill=1)
    # faixa de severidades no topo
    x = 0
    por_sev = contar_por_severidade()
    total = sum(por_sev.values()) or 1
    for sev, qtd in por_sev.items():
        if not qtd:
            continue
        w = largura * (qtd / total)
        canvas.setFillColor(C[sev])
        canvas.rect(x, altura - 6.3 * cm, w, 0.32 * cm, stroke=0, fill=1)
        x += w
    canvas.setFont("Helvetica-Bold", 8.6)
    canvas.setFillColor(colors.HexColor("#94A3B8"))
    canvas.drawString(2 * cm, altura - 2.15 * cm, "AUDITORIA DE SEGURANÇA DE CÓDIGO")
    canvas.setFont("Helvetica-Bold", 24)
    canvas.setFillColor(colors.white)
    canvas.drawString(2 * cm, altura - 3.35 * cm, "Relatorio de Auditoria")
    canvas.drawString(2 * cm, altura - 4.45 * cm, "de Seguranca")
    canvas.setFont("Helvetica", 15)
    canvas.setFillColor(colors.HexColor("#38BDF8"))
    canvas.drawString(2 * cm, altura - 5.45 * cm, A.PROJETO)
    # rodape da capa
    canvas.setStrokeColor(LINHA)
    canvas.setLineWidth(0.6)
    canvas.line(2 * cm, 1.42 * cm, largura - 2 * cm, 1.42 * cm)
    canvas.setFont("Helvetica", 7.6)
    canvas.setFillColor(TINTA_SUAVE)
    canvas.drawString(2 * cm, 1.02 * cm, "%s - commit %s" % (A.PROJETO, A.COMMIT))
    canvas.drawRightString(largura - 2 * cm, 1.02 * cm, "Página %d" % doc.page)
    canvas.restoreState()

    # --------------------------------------------------------------------------
    # Blocos de conteudo
    # --------------------------------------------------------------------------


def bloco_capa(e):
    fluxo = [Spacer(1, 5.0 * cm)]

    meta = [
        ["Data da auditoria", A.DATA_AUDITORIA],
        ["Commit auditado", A.COMMIT],
        ["Branch", A.BRANCH],
        ["Total de achados", "%d (ver distribuição no resumo executivo)" % len(A.ACHADOS)],
        ["Pontos fortes verificados", "%d" % len(A.PONTOS_FORTES)],
    ]
    t = Table(
        [
            [Paragraph("<b>%s</b>" % esc(k), e["celula"]), Paragraph(esc(v), e["celula"])]
            for k, v in meta
        ],
        colWidths=[4.6 * cm, 12.4 * cm],
    )
    t.setStyle(
        TableStyle(
            [
                ("VALIGN", (0, 0), (-1, -1), "TOP"),
                ("LINEBELOW", (0, 0), (-1, -2), 0.4, LINHA),
                ("TOPPADDING", (0, 0), (-1, -1), 5),
                ("BOTTOMPADDING", (0, 0), (-1, -1), 5),
                ("LEFTPADDING", (0, 0), (-1, -1), 0),
            ]
        )
    )
    fluxo += [t, Spacer(1, 0.7 * cm)]

    fluxo.append(Paragraph("Escopo auditado", e["h2"]))
    fluxo.append(
        Paragraph(
            "Auditoria manual, arquivo por arquivo e linha por linha, de todo o código-fonte Go do "
            "repositório (79 arquivos, ~35.800 linhas), da configuração de build e empacotamento "
            "(Dockerfile, .dockerignore, Makefile), dos 5 workflows do GitHub Actions, das configurações "
            "de ferramentas de segurança (.whitesource, dependabot.yml), da documentação (README, docs/) e "
            "do histórico Git completo (75 commits). Não foram auditadas as dependências de terceiros em "
            "si, já cobertas por govulncheck, Trivy e Mend no CI, nem os diretórios de artefatos de "
            "agentes (_reversa_sdd/, .specsmd/, .specify/, specs/), que não integram o binário.",
            e["corpo"],
        )
    )

    fluxo.append(Paragraph("Nota metodologica", e["h2"]))
    fluxo.append(
        Paragraph(
            "As cinco categorias solicitadas foram formuladas para aplicações web com banco de dados, "
            "autenticação e frontend. Este projeto é uma <b>ferramenta de linha de comando local</b>, sem "
            "servidor, sem banco, sem usuários e sem interface web. Em vez de forçar achados onde as "
            "categorias não se aplicam, cada uma foi mapeada para o seu equivalente estrutural nesta stack, "
            "e a não aplicabilidade é declarada de forma explícita. O detalhamento do mapeamento está na "
            "secao 1.",
            e["corpo"],
        )
    )
    fluxo.append(
        Paragraph(
            "Todo achado registrado foi verificado no código real, com caminho de arquivo e número de "
            "linha. O achado F1 foi adicionalmente confirmado por execução de um teste ad hoc contra a "
            "API real do projeto. Nenhum achado é especulativo.",
            e["nota"],
        )
    )
    return fluxo


def bloco_stack(e):
    fluxo = [Paragraph("1. Stack detectada e mapeamento das categorias", e["h1"])]
    fluxo.append(Paragraph("1.1 Stack detectada", e["h2"]))

    linhas = [
        [Paragraph("<b>%s</b>" % esc(k), e["celula"]), Paragraph(esc(v), e["celula"])]
        for k, v in A.STACK.items()
    ]
    t = Table(linhas, colWidths=[3.9 * cm, 13.1 * cm], repeatRows=0)
    t.setStyle(
        TableStyle(
            [
                ("VALIGN", (0, 0), (-1, -1), "TOP"),
                ("ROWBACKGROUNDS", (0, 0), (-1, -1), [colors.white, FUNDO_SUAVE]),
                ("LINEBELOW", (0, 0), (-1, -2), 0.35, LINHA),
                ("BOX", (0, 0), (-1, -1), 0.5, LINHA),
                ("TOPPADDING", (0, 0), (-1, -1), 5),
                ("BOTTOMPADDING", (0, 0), (-1, -1), 5),
                ("LEFTPADDING", (0, 0), (-1, -1), 6),
                ("RIGHTPADDING", (0, 0), (-1, -1), 6),
            ]
        )
    )
    fluxo += [t, Spacer(1, 0.35 * cm)]

    fluxo.append(Paragraph("1.2 Como cada categoria foi mapeada", e["h2"]))
    for cat in A.MAPEAMENTO_CATEGORIAS:
        marca = "APLICAVEL" if cat["aplicavel"] else "NÃO SE APLICA DIRETAMENTE"
        cor = C["forte"] if cat["aplicavel"] else TINTA_SUAVE
        cab = Table(
            [
                [
                    Paragraph(
                        "<b>Categoria %d - %s</b>" % (cat["n"], esc(cat["titulo"])), e["celula"]
                    ),
                    Paragraph(
                        '<font color="%s"><b>%s</b></font>' % (cor.hexval()[2:], marca), e["celula"]
                    ),
                ]
            ],
            colWidths=[12.2 * cm, 4.8 * cm],
        )
        cab.setStyle(
            TableStyle(
                [
                    ("VALIGN", (0, 0), (-1, -1), "MIDDLE"),
                    ("BACKGROUND", (0, 0), (-1, -1), FUNDO_SUAVE),
                    ("ALIGN", (1, 0), (1, 0), "RIGHT"),
                    ("TOPPADDING", (0, 0), (-1, -1), 5),
                    ("BOTTOMPADDING", (0, 0), (-1, -1), 5),
                    ("LEFTPADDING", (0, 0), (-1, -1), 6),
                    ("RIGHTPADDING", (0, 0), (-1, -1), 6),
                ]
            )
        )
        fluxo.append(
            KeepTogether(
                [
                    cab,
                    Spacer(1, 0.12 * cm),
                    Paragraph(esc(cat["mapeamento"]), e["corpo"]),
                    Spacer(1, 0.18 * cm),
                ]
            )
        )
    return fluxo


def bloco_resumo(e, graf_rosca, graf_barras):
    por_sev = contar_por_severidade()
    fluxo = [PageBreak(), Paragraph("2. Resumo executivo", e["h1"])]

    # KPIs coloridos
    celulas, larguras = [], []
    for sev in ORDEM_SEV:
        qtd = por_sev[sev]
        inner = Table(
            [
                [Paragraph(str(qtd), e["kpi_num"])],
                [Paragraph(A.ROTULO_SEVERIDADE[sev], e["kpi_lbl"])],
            ],
            colWidths=[3.15 * cm],
        )
        inner.setStyle(
            TableStyle(
                [
                    ("BACKGROUND", (0, 0), (-1, -1), C[sev]),
                    ("VALIGN", (0, 0), (-1, -1), "MIDDLE"),
                    ("TOPPADDING", (0, 0), (-1, 0), 9),
                    ("BOTTOMPADDING", (0, 1), (-1, 1), 9),
                    ("TOPPADDING", (0, 1), (-1, 1), 0),
                    ("BOTTOMPADDING", (0, 0), (-1, 0), 1),
                ]
            )
        )
        celulas.append(inner)
        larguras.append(3.3 * cm)
    kpi = Table([celulas], colWidths=larguras)
    kpi.setStyle(
        TableStyle(
            [
                ("VALIGN", (0, 0), (-1, -1), "MIDDLE"),
                ("LEFTPADDING", (0, 0), (-1, -1), 0),
                ("RIGHTPADDING", (0, 0), (-1, -1), 4),
            ]
        )
    )
    fluxo += [kpi, Spacer(1, 0.45 * cm)]

    fluxo.append(
        Paragraph(
            "A auditoria registrou <b>%d achados</b> e <b>%d pontos fortes verificados</b>. Não há nenhuma "
            "vulnerabilidade de severidade crítica: o projeto não expoe superfície de rede, não possui "
            "segredos hardcoded e implementa corretamente as primitivas criptográficas do KeyStore V3. "
            "Os riscos concentram-se em <b>como o material criptográfico é gerado e persistido</b> - "
            "quatro achados de severidade alta, todos no caminho que produz e grava chaves privadas."
            % (len(A.ACHADOS), len(A.PONTOS_FORTES)),
            e["corpo"],
        )
    )

    fluxo.append(Spacer(1, 0.2 * cm))
    fluxo.append(Paragraph("2.1 Distribuição por severidade", e["h2"]))
    fluxo.append(Image(graf_rosca, width=9.0 * cm, height=6.4 * cm, hAlign="CENTER"))
    fluxo.append(Spacer(1, 0.35 * cm))
    fluxo.append(Paragraph("2.2 Distribuição por categoria", e["h2"]))
    fluxo.append(
        Paragraph(
            "Cada barra é colorida pela severidade mais alta encontrada naquela categoria.",
            e["nota"],
        )
    )
    fluxo.append(Image(graf_barras, width=15.4 * cm, height=7.9 * cm, hAlign="CENTER"))
    return fluxo


def bloco_fortes_fracos(e):
    fluxo = [PageBreak(), Paragraph("3. Pontos fortes", e["h1"])]
    fluxo.append(
        Paragraph(
            "O que foi verificado e está correto. Esta seção também serve de prova de cobertura da "
            "auditoria: cada item corresponde a um caminho de código efetivamente percorrido.",
            e["corpo"],
        )
    )

    for pf in A.PONTOS_FORTES:
        cab = Table(
            [
                [
                    Paragraph(
                        '<font color="%s"><b>%s</b></font>  %s'
                        % (C["forte"].hexval()[2:], pf["id"], esc(pf["titulo"])),
                        e["celula"],
                    )
                ]
            ],
            colWidths=[17.0 * cm],
        )
        cab.setStyle(
            TableStyle(
                [
                    ("BACKGROUND", (0, 0), (-1, -1), colors.HexColor("#ECFDF5")),
                    ("LINEBEFORE", (0, 0), (0, -1), 2.4, C["forte"]),
                    ("TOPPADDING", (0, 0), (-1, -1), 5),
                    ("BOTTOMPADDING", (0, 0), (-1, -1), 5),
                    ("LEFTPADDING", (0, 0), (-1, -1), 7),
                    ("RIGHTPADDING", (0, 0), (-1, -1), 6),
                ]
            )
        )
        fluxo.append(
            KeepTogether(
                [
                    cab,
                    Spacer(1, 0.1 * cm),
                    Paragraph(esc(pf["evidencia"]), e["corpo"]),
                    Spacer(1, 0.16 * cm),
                ]
            )
        )

    fluxo.append(PageBreak())
    fluxo.append(Paragraph("4. Pontos fracos - riscos centrais", e["h1"]))
    riscos = [
        (
            "As chaves privadas entregues não são independentes entre si (F1)",
            "Na engine CPU, que é o caminho padrão fora do macOS ARM, as carteiras geradas em uma mesma "
            "execução são escalares consecutivos. Comprometer uma compromete até 4095 outras. É o único achado que ataca diretamente a premissa fundamental de um gerador de carteiras, é o único "
            "que não está documentado em lugar nenhum.",
        ),
        (
            "O layout em disco anula a criptografia que o próprio projeto implementa corretamente (F2, F3)",
            "O keystore V3 está criptograficamente correto - scrypt N=262144, AES-128-CTR, MAC Keccak256, "
            "comparação em tempo constante. Mas a senha que o abre fica em texto claro no arquivo ao lado, "
            "e em Solana a chave é gravada sem cifra alguma. Copiar o diretório equivale a copiar as "
            "chaves privadas.",
        ),
        (
            "Um backup que não restaura (F4)",
            "Em Bitcoin, o único artefato persistido é uma mnemônica sem relação matemática com a chave "
            "gerada. A chave real só aparece no stdout. É uma falha de perda de material com impacto "
            "financeiro direto é irreversível, agravada por um arquivo que aparenta ser um backup válido.",
        ),
        (
            "Controles de segurança declarados que não chegam a camada que executa (F5, F8, F10)",
            "--security-level, --kdf-params, keystore.file_mode, --output e --format são aceitos, "
            "validados e descartados. Nenhum deles hoje causa dano - todos falham de forma segura - mas "
            "juntos criam uma discrepância entre a política que o operador acredita ter aplicado e a que "
            "está em vigor, e uma armadilha de regressão (ver a ordem de correção em P2).",
        ),
        (
            "Superfície de build mais permissiva que a superfície de execução (F9, F12)",
            "O binário é cuidadoso; o pipeline que o publica não. Há injeção de script num workflow com "
            "permissão de escrita em releases e pacotes, uma action de terceiro em referência mutável, e "
            "o Dependabot nunca chegou a ser configurado.",
        ),
    ]
    for titulo, texto in riscos:
        fluxo.append(Paragraph(esc(titulo), e["h2"]))
        fluxo.append(Paragraph(esc(texto), e["corpo"]))
    return fluxo


def bloco_achados(e):
    fluxo = [PageBreak(), Paragraph("5. Achados detalhados", e["h1"])]

    # tabela sintetica
    fluxo.append(Paragraph("5.1 Síntese", e["h2"]))
    cab = [
        Paragraph("Sev.", e["celula_cab"]),
        Paragraph("ID", e["celula_cab"]),
        Paragraph("Arquivo:linha", e["celula_cab"]),
        Paragraph("Descrição", e["celula_cab"]),
    ]
    linhas = [cab]
    estilo_tab = [
        ("VALIGN", (0, 0), (-1, -1), "TOP"),
        ("BACKGROUND", (0, 0), (-1, 0), colors.HexColor("#0F172A")),
        ("TEXTCOLOR", (0, 0), (-1, 0), colors.white),
        ("BOX", (0, 0), (-1, -1), 0.5, LINHA),
        ("INNERGRID", (0, 0), (-1, -1), 0.3, LINHA),
        ("TOPPADDING", (0, 0), (-1, -1), 4),
        ("BOTTOMPADDING", (0, 0), (-1, -1), 4),
        ("LEFTPADDING", (0, 0), (-1, -1), 5),
        ("RIGHTPADDING", (0, 0), (-1, -1), 5),
    ]
    for i, a in enumerate(A.ACHADOS, start=1):
        linhas.append(
            [
                chip_severidade(a["severidade"], e),
                Paragraph("<b>%s</b>" % a["id"], e["celula"]),
                Paragraph(
                    "<br/>".join(esc(x) for x in a["arquivos"][:4])
                    + ("<br/>(+%d)" % (len(a["arquivos"]) - 4) if len(a["arquivos"]) > 4 else ""),
                    e["celula_mono"],
                ),
                Paragraph(esc(a["titulo"]), e["celula"]),
            ]
        )
        estilo_tab.append(("TEXTCOLOR", (1, i), (1, i), C[a["severidade"]]))
    t = Table(linhas, colWidths=[2.25 * cm, 0.95 * cm, 6.6 * cm, 7.2 * cm], repeatRows=1)
    t.setStyle(TableStyle(estilo_tab))
    fluxo += [t, Spacer(1, 0.3 * cm)]

    # detalhe por achado, agrupado por categoria
    fluxo.append(PageBreak())
    fluxo.append(Paragraph("5.2 Detalhamento por categoria", e["h2"]))
    for cat in contar_por_categoria().keys():
        fluxo.append(Paragraph(esc(cat), e["h2"]))
        for a in [x for x in A.ACHADOS if x["categoria"] == cat]:
            cab = Table(
                [
                    [
                        chip_severidade(a["severidade"], e),
                        Paragraph("<b>%s - %s</b>" % (a["id"], esc(a["titulo"])), e["celula"]),
                    ]
                ],
                colWidths=[2.3 * cm, 14.7 * cm],
            )
            cab.setStyle(
                TableStyle(
                    [
                        ("VALIGN", (0, 0), (-1, -1), "MIDDLE"),
                        ("BACKGROUND", (1, 0), (1, 0), FUNDO_SUAVE),
                        ("LEFTPADDING", (0, 0), (0, 0), 0),
                        ("LEFTPADDING", (1, 0), (1, 0), 7),
                        ("RIGHTPADDING", (0, 0), (-1, -1), 5),
                        ("TOPPADDING", (0, 0), (-1, -1), 4),
                        ("BOTTOMPADDING", (0, 0), (-1, -1), 4),
                    ]
                )
            )
            fluxo.append(KeepTogether([cab, Spacer(1, 0.14 * cm)]))

            fluxo.append(Paragraph("Localização", e["h3"]))
            fluxo.append(Paragraph("<br/>".join(esc(x) for x in a["arquivos"]), e["celula_mono"]))
            fluxo.append(Spacer(1, 0.16 * cm))

            fluxo.append(Paragraph("Evidência no código", e["h3"]))
            fluxo.append(
                Paragraph(
                    esc(a["trecho"]).replace("\n", "<br/>").replace(" ", "&nbsp;"), e["codigo"]
                )
            )

            fluxo.append(Paragraph("Por que é explorável", e["h3"]))
            for par in a["porque"].split("\n\n"):
                fluxo.append(Paragraph(esc(par), e["corpo"]))

            fluxo.append(Paragraph("Condições de explorabilidade", e["h3"]))
            fluxo.append(Paragraph(esc(a["condicoes"]), e["nota"]))
            fluxo.append(Spacer(1, 0.3 * cm))
    return fluxo


def bloco_recomendacoes(e):
    fluxo = [PageBreak(), Paragraph("6. Recomendações priorizadas", e["h1"])]
    cor_prio = {"P1": C["critica"], "P2": C["media"], "P3": C["baixa"]}
    for r in A.RECOMENDACOES:
        cor = cor_prio.get(r["prioridade"], TINTA_SUAVE)
        badge = Table(
            [[Paragraph(r["prioridade"], e["chip"])]], colWidths=[1.15 * cm], rowHeights=[0.52 * cm]
        )
        badge.setStyle(
            TableStyle(
                [
                    ("BACKGROUND", (0, 0), (-1, -1), cor),
                    ("VALIGN", (0, 0), (-1, -1), "MIDDLE"),
                    ("LEFTPADDING", (0, 0), (-1, -1), 1),
                    ("RIGHTPADDING", (0, 0), (-1, -1), 1),
                    ("TOPPADDING", (0, 0), (-1, -1), 1),
                    ("BOTTOMPADDING", (0, 0), (-1, -1), 1),
                ]
            )
        )
        cab = Table(
            [
                [
                    badge,
                    Paragraph(
                        "<b>%s</b><br/><font size=7.6 color='#475569'>Achados: %s</font>"
                        % (esc(r["titulo"]), esc(r["achados"])),
                        e["celula"],
                    ),
                ]
            ],
            colWidths=[1.4 * cm, 15.6 * cm],
        )
        cab.setStyle(
            TableStyle(
                [
                    ("VALIGN", (0, 0), (-1, -1), "MIDDLE"),
                    ("BACKGROUND", (1, 0), (1, 0), FUNDO_SUAVE),
                    ("LEFTPADDING", (0, 0), (0, 0), 0),
                    ("LEFTPADDING", (1, 0), (1, 0), 7),
                    ("TOPPADDING", (0, 0), (-1, -1), 4),
                    ("BOTTOMPADDING", (0, 0), (-1, -1), 4),
                ]
            )
        )
        fluxo.append(
            KeepTogether(
                [
                    cab,
                    Spacer(1, 0.12 * cm),
                    Paragraph(esc(r["detalhe"]), e["corpo"]),
                    Spacer(1, 0.2 * cm),
                ]
            )
        )
    return fluxo


def bloco_issues(e):
    fluxo = [PageBreak(), Paragraph("7. Issues para o GitHub", e["h1"])]
    fluxo.append(
        Paragraph(
            "Texto completo de cada issue em Markdown, pronto para copiar e colar. Achados triviais "
            "relacionados foram agrupados numa mesma issue para evitar ruído: os %d achados desta auditoria "
            "resultam em %d issues acionáveis." % (len(A.ACHADOS), len(A.ISSUES)),
            e["corpo"],
        )
    )

    for it in A.ISSUES:
        md = montar_markdown_issue(it)
        fluxo.append(Paragraph("--- ISSUE %d ---" % it["n"], e["issue_cab"]))
        # Cada linha vira um Paragraph proprio: o bloco pode quebrar entre páginas
        # (uma Table de linha única não quebraria e estouraria o frame), e os
        # fundos consecutivos se emendam visualmente numa unica faixa.
        for linha in md.split("\n"):
            if linha.strip() == "":
                fluxo.append(Paragraph("&nbsp;", e["issue_md"]))
            else:
                # preserva apenas a indentacao inicial: o resto quebra por palavra
                recuo = len(linha) - len(linha.lstrip(" "))
                fluxo.append(
                    Paragraph("&nbsp;" * recuo + esc(linha.lstrip(" ")), e["issue_md"])
                )
        fluxo.append(Paragraph("--- FIM ISSUE %d ---" % it["n"], e["issue_cab"]))
    return fluxo


def montar_markdown_issue(it):
    partes = []
    partes.append("**Titulo:** %s" % it["titulo"])
    partes.append("")
    partes.append("**Labels:** `%s`" % "`, `".join(x.strip() for x in it["labels"].split(",")))
    partes.append("")
    partes.append("## Problema")
    partes.append("")
    partes.append(it["problema"])
    partes.append("")
    partes.append("## Evidência")
    partes.append("")
    partes.append(it["evidencia"])
    partes.append("")
    partes.append("## Impacto")
    partes.append("")
    partes.append(it["impacto"])
    partes.append("")
    partes.append("## Sugestão de correção")
    partes.append("")
    partes.append(it["correcao"])
    partes.append("")
    partes.append("## Critérios de aceite")
    partes.append("")
    for c in it["criterios"]:
        partes.append("- [ ] %s" % c)
    return "\n".join(partes)

    # --------------------------------------------------------------------------
    # Markdown completo
    # --------------------------------------------------------------------------


def gerar_markdown():
    por_sev = contar_por_severidade()
    por_cat = contar_por_categoria()
    L = []
    a = L.append

    a("# Relatório de Auditoria de Segurança - %s" % A.PROJETO)
    a("")
    a("| | |")
    a("|---|---|")
    a("| **Data da auditoria** | %s |" % A.DATA_AUDITORIA)
    a("| **Commit auditado** | `%s` |" % A.COMMIT)
    a("| **Branch** | `%s` |" % A.BRANCH)
    a("| **Total de achados** | %d |" % len(A.ACHADOS))
    a("| **Pontos fortes verificados** | %d |" % len(A.PONTOS_FORTES))
    a("")
    a("## Escopo auditado")
    a("")
    a(
        "Auditoria manual, arquivo por arquivo e linha por linha, de todo o código-fonte Go do "
        "repositório (79 arquivos, ~35.800 linhas), da configuração de build e empacotamento "
        "(Dockerfile, .dockerignore, Makefile), dos 5 workflows do GitHub Actions, das configurações de "
        "ferramentas de segurança (.whitesource, dependabot.yml), da documentação (README, docs/) e do "
        "histórico Git completo (75 commits). Não foram auditadas as dependências de terceiros em si, já "
        "cobertas por govulncheck, Trivy e Mend no CI, nem os diretórios de artefatos de agentes "
        "(`_reversa_sdd/`, `.specsmd/`, `.specify/`, `specs/`), que não integram o binário."
    )
    a("")
    a("## Nota metodológica")
    a("")
    a(
        "As cinco categorias solicitadas foram formuladas para aplicações web com banco de dados, "
        "autenticação e frontend. Este projeto é uma **ferramenta de linha de comando local**, sem "
        "servidor, sem banco, sem usuários e sem interface web. Em vez de forçar achados onde as "
        "categorias não se aplicam, cada uma foi mapeada para o seu equivalente estrutural nesta stack, "
        "e a não aplicabilidade é declarada de forma explícita."
    )
    a("")
    a(
        "Todo achado registrado foi verificado no código real, com caminho de arquivo e número de linha. "
        "O achado F1 foi adicionalmente confirmado por execução de um teste ad hoc contra a API real do "
        "projeto. Nenhum achado é especulativo."
    )
    a("")

    a("## 1. Stack detectada e mapeamento das categorias")
    a("")
    a("### 1.1 Stack detectada")
    a("")
    a("| Aspecto | Detectado |")
    a("|---|---|")
    for k, v in A.STACK.items():
        a("| **%s** | %s |" % (k, v.replace("|", "\\|")))
    a("")
    a("### 1.2 Como cada categoria foi mapeada")
    a("")
    for cat in A.MAPEAMENTO_CATEGORIAS:
        marca = "APLICAVEL" if cat["aplicavel"] else "NÃO SE APLICA DIRETAMENTE"
        a("#### Categoria %d - %s" % (cat["n"], cat["titulo"]))
        a("")
        a("**Status:** %s" % marca)
        a("")
        a(cat["mapeamento"])
        a("")

    a("## 2. Resumo executivo")
    a("")
    a("| Severidade | Achados |")
    a("|---|---|")
    for sev in ORDEM_SEV:
        a("| %s | %d |" % (A.ROTULO_SEVERIDADE[sev].title(), por_sev[sev]))
    a("| **Total** | **%d** |" % len(A.ACHADOS))
    a("")
    a("| Categoria | Achados | Severidade mais alta |")
    a("|---|---|---|")
    for cat, qtd in por_cat.items():
        a("| %s | %d | %s |" % (cat, qtd, A.ROTULO_SEVERIDADE[pior_severidade(cat)].title()))
    a("")
    a("Gráficos: `gráficos/severidade-rosca.png` e `gráficos/categoria-barras.png`.")
    a("")
    a(
        "Não há nenhuma vulnerabilidade de severidade crítica: o projeto não expoe superfície de rede, "
        "não possui segredos hardcoded e implementa corretamente as primitivas criptográficas do "
        "KeyStore V3. Os riscos concentram-se em **como o material criptográfico é gerado e "
        "persistido** - quatro achados de severidade alta, todos no caminho que produz e grava chaves "
        "privadas."
    )
    a("")

    a("## 3. Pontos fortes")
    a("")
    a("O que foi verificado e está correto. Serve também de prova de cobertura da auditoria.")
    a("")
    for pf in A.PONTOS_FORTES:
        a("### %s - %s" % (pf["id"], pf["titulo"]))
        a("")
        a(pf["evidencia"])
        a("")

    a("## 4. Pontos fracos - riscos centrais")
    a("")
    for titulo, texto in [
        (
            "As chaves privadas entregues não são independentes entre si (F1)",
            "Na engine CPU, que é o caminho padrão fora do macOS ARM, as carteiras geradas em uma mesma "
            "execução são escalares consecutivos. Comprometer uma compromete até 4095 outras. É o único achado que ataca diretamente a premissa fundamental de um gerador de carteiras, é o único "
            "que não está documentado em lugar nenhum.",
        ),
        (
            "O layout em disco anula a criptografia que o próprio projeto implementa corretamente (F2, F3)",
            "O keystore V3 está criptograficamente correto - scrypt N=262144, AES-128-CTR, MAC Keccak256, "
            "comparação em tempo constante. Mas a senha que o abre fica em texto claro no arquivo ao lado, "
            "e em Solana a chave é gravada sem cifra alguma. Copiar o diretório equivale a copiar as "
            "chaves privadas.",
        ),
        (
            "Um backup que não restaura (F4)",
            "Em Bitcoin, o único artefato persistido é uma mnemônica sem relação matemática com a chave "
            "gerada. A chave real só aparece no stdout. É uma falha de perda de material com impacto "
            "financeiro direto é irreversível.",
        ),
        (
            "Controles de segurança declarados que não chegam a camada que executa (F5, F8, F10)",
            "`--security-level`, `--kdf-params`, `keystore.file_mode`, `--output` e `--format` são "
            "aceitos, validados e descartados. Nenhum causa dano hoje - todos falham de forma segura - "
            "mas criam discrepância entre política declarada e aplicada, e uma armadilha de regressão.",
        ),
        (
            "Superfície de build mais permissiva que a superfície de execução (F9, F12)",
            "O binário é cuidadoso; o pipeline que o publica não. Injeção de script num workflow com "
            "permissão de escrita em releases e pacotes, action de terceiro em referência mutável, e "
            "Dependabot nunca configurado.",
        ),
    ]:
        a("### %s" % titulo)
        a("")
        a(texto)
        a("")

    a("## 5. Achados detalhados")
    a("")
    a("### 5.1 Síntese")
    a("")
    a("| Severidade | ID | Arquivo:linha | Descrição |")
    a("|---|---|---|---|")
    for f in A.ACHADOS:
        arqs = "<br>".join("`%s`" % x for x in f["arquivos"])
        a(
            "| **%s** | %s | %s | %s |"
            % (
                A.ROTULO_SEVERIDADE[f["severidade"]].title(),
                f["id"],
                arqs,
                f["titulo"].replace("|", "\\|"),
            )
        )
    a("")
    a("### 5.2 Detalhamento por categoria")
    a("")
    for cat in por_cat.keys():
        a("### %s" % cat)
        a("")
        for f in [x for x in A.ACHADOS if x["categoria"] == cat]:
            a("#### %s - %s" % (f["id"], f["titulo"]))
            a("")
            a("**Severidade:** %s" % A.ROTULO_SEVERIDADE[f["severidade"]].title())
            a("")
            a("**Localização:**")
            a("")
            for x in f["arquivos"]:
                a("- `%s`" % x)
            a("")
            a("**Evidência no código:**")
            a("")
            a("```go")
            a(f["trecho"])
            a("```")
            a("")
            a("**Por que é explorável:**")
            a("")
            a(f["porque"])
            a("")
            a("**Condições de explorabilidade:**")
            a("")
            a(f["condicoes"])
            a("")

    a("## 6. Recomendações priorizadas")
    a("")
    for r in A.RECOMENDACOES:
        a("### %s - %s" % (r["prioridade"], r["titulo"]))
        a("")
        a("**Achados cobertos:** %s" % r["achados"])
        a("")
        a(r["detalhe"])
        a("")

    a("## 7. Issues para o GitHub")
    a("")
    a(
        "Texto completo de cada issue em Markdown, pronto para copiar e colar. Achados triviais "
        "relacionados foram agrupados numa mesma issue: os %d achados resultam em %d issues acionáveis."
        % (len(A.ACHADOS), len(A.ISSUES))
    )
    a("")
    for it in A.ISSUES:
        a("--- ISSUE %d ---" % it["n"])
        a("")
        a(montar_markdown_issue(it))
        a("")
        a("--- FIM ISSUE %d ---" % it["n"])
        a("")

    with open(MD_PATH, "w", encoding="utf-8") as fh:
        fh.write("\n".join(L) + "\n")
    return MD_PATH

    # --------------------------------------------------------------------------
    # Montagem do PDF
    # --------------------------------------------------------------------------


def gerar_pdf(graf_rosca, graf_barras):
    e = construir_estilos()
    doc = BaseDocTemplate(
        PDF_PATH,
        pagesize=A4,
        leftMargin=2 * cm,
        rightMargin=2 * cm,
        topMargin=2 * cm,
        bottomMargin=2 * cm,
        title=TITULO_RELATORIO,
        author="Auditoria de seguranca",
        subject="Seguranca de aplicacao",
    )

    frame_capa = Frame(2 * cm, 2 * cm, A4[0] - 4 * cm, A4[1] - 4 * cm, id="capa")
    frame_corpo = Frame(2 * cm, 2 * cm, A4[0] - 4 * cm, A4[1] - 4 * cm, id="corpo")
    doc.addPageTemplates(
        [
            PageTemplate(id="Capa", frames=[frame_capa], onPage=desenhar_capa),
            PageTemplate(id="Corpo", frames=[frame_corpo], onPage=desenhar_moldura),
        ]
    )

    fluxo = []
    fluxo += bloco_capa(e)
    fluxo.append(NextPageTemplate("Corpo"))
    fluxo.append(PageBreak())
    fluxo += bloco_stack(e)
    fluxo += bloco_resumo(e, graf_rosca, graf_barras)
    fluxo += bloco_fortes_fracos(e)
    fluxo += bloco_achados(e)
    fluxo += bloco_recomendacoes(e)
    fluxo += bloco_issues(e)

    doc.build(fluxo)
    return PDF_PATH


def main():
    rosca, barras = gerar_graficos()
    pdf = gerar_pdf(rosca, barras)
    md = gerar_markdown()
    print("PDF: %s" % pdf)
    print("Markdown: %s" % md)
    print("Graficos: %s, %s" % (rosca, barras))


if __name__ == "__main__":
    main()
