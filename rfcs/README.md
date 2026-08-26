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
| [if_and_call.md](if_and_call.md) | proposta | como `if` e `call` viram bytecode, e a recursão que sai do segundo |
| [ir.md](ir.md) | proposta | o IR é a fita do evaluator, e devia descrever o programa — pré-requisito da anterior |
| [folding_literals.md](folding_literals.md) | proposta | um terço do IR existe para nomear números já conhecidos, e o que dobrar isso exige |
| [effect.md](effect.md) | proposta | o IR não diz o que uma instrução faz além de deixar um valor, e o builder move instrução |

A última foi `shape_inference.md` — a linguagem descobre qual shape um escopo retorna, e o
`returns` vira uma restrição que o compilador cumpre em vez de ser a única fonte. Foi
implementada em cinco pull requests: três de vocabulário, porque a mesma coisa estava sendo
chamada de três nomes e isso teve que ser resolvido antes de escrever mais sobre ela, e dois
de comportamento. O que ficou decidido está em
[docs/language-design.md](../docs/language-design.md), na seção "Declaring a shape with
`returns`", e os três lugares onde a descoberta para estão no
[roadmap](../docs/roadmap.md), na seção "Shapes".

Antes dela, `crossing_shapes.md` — a forma de um struct atravessa módulo, e o nome dele é
escrito como o do módulo. Foi implementada em dois pull requests, e o que ficou decidido está
em [docs/modules.md](../docs/modules.md). O que ela decidiu antes de tudo foi **em que ordem
ler os arquivos**: dependência primeiro, como Go e Rust, e não parseia-tudo-resolve-depois como
Java e TypeScript — porque os dois adiam para um alvo que carrega nomes, e a EVM não carrega.

Antes dela, `returns.md` — um bloco promete a forma que responde, e o compilador recusa o
bloco que não cumpre. Foi implementada no mesmo dia em que foi escrita, e o que ficou decidido
está em [docs/language-design.md](../docs/language-design.md), na seção "Declaring a shape with
`returns`" — incluindo o que ela decidiu **contra**: o `returns` não é exigido, e não será nunca,
porque um escopo não tem assinatura e exigir que ele declare o que responde seria declarar uma
ponta e não a outra.

Antes dela, `module_system.md` — um arquivo é um módulo, um nome carrega o módulo a que
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
