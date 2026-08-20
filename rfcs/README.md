# RFCs

Propostas, dívidas e discussões em aberto sobre a Aurora — **em português**, que é a língua em
que o projeto é discutido. O que já está decidido e implementado mora em `docs/`, em inglês;
aqui fica o que ainda está sendo pensado.

Uma RFC não é um roadmap. O [roadmap](../docs/roadmap.md) diz **o que** falta e o que isso
custa; uma RFC discute **como** resolver uma coisa específica, quais são as alternativas e por
que uma foi escolhida. Uma proposta aceita e implementada vira documentação em `docs/` e o
arquivo daqui é apagado — o git guarda o histórico.

Toda RFC abre dizendo em que estado está:

| Estado | Significa |
|---|---|
| **proposta** | escrita, ainda não decidida |
| **aceita** | decidida, ainda não implementada |
| **implementada** | virou código; o arquivo sai daqui na sequência |
| **recusada** | decidida contra, e o porquê fica registrado |

**Abertas:**

| RFC | Estado | Sobre |
|---|---|---|
| [returns.md](returns.md) | proposta | `returns`: um escopo promete a forma que responde, e o compilador confere |

A última foi `module_system.md` — um arquivo é um módulo, um nome carrega o módulo a que
pertence, e cada módulo guarda os seus nomes num environ próprio. Foi implementada em seis
pull requests e o que ficou decidido está em [docs/modules.md](../docs/modules.md), que é a
referência, e em [docs/module_system_design.md](../docs/module_system_design.md), que é o
porquê — incluindo o que foi recusado e o argumento que a própria RFC teve que abandonar no
meio do caminho. O arquivo saiu daqui como a regra acima manda; o git guarda a discussão
inteira.

Antes dela, `phase_coupling.md` — fases não se conhecem: artefatos viram pacotes de
`wire`, a montagem sobe para o `main`. Foi implementada inteira e o que ficou decidido está em
[docs/contributing/architecture.md](../docs/contributing/architecture.md), que é a fonte da
verdade sobre a estrutura do código. O arquivo saiu daqui como a regra acima manda; o git
guarda o caminho, incluindo o que foi descoberto ao aplicar cada etapa.
