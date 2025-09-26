|  General Info  | |
| ---|---|
| Working Title | SOMACS v0.1 |
| Created By | Tobias Buhl, tobias.buhl@stud-mail.uni-wuerzburg.de |
| Target Platform(s) | Windows 10+ |
| Engine Version | go 1.23.2, GoLand2024.2.3 |

### Abstract

A framework for complexity reduction-based surrogate modelling of self-organising multi-agent systems. 
It follows the concept middle-out abstraction: Observer agents listen to model agents which are then subsumed by meta agents.
Additionally, it is built upon the base platform SOMAS (https://github.com/MattSScott/basePlatformSOMAS).

To get started:
- Open the project in GoLand or another go environment. If it doesn't happen automatically, download the packages in the go.sum
- Run Example.go to test the framework. Setting explainabilityVerbose=true showcases the evaluate and explainability functions.
- Implement your own simulation using the patterns shown in Example.go - the framework code aims to be self-documenting, an official documentation will follow in an upcoming version

### Paper Abstract

Surrogate models, i.e. approximation-based emulators, can offer groundbreaking advantages, including reduction of computational cost at runtime and built-in explainability. In the field of self-organisation, complexity reduction seems especially promising, i.e. utilising emergence to describe parts of the system as a group rather than as individuals. In this paper, we take a step towards facilitating the integration of surrogate modelling into self-organising system simulations.
The resulting framework (SOMACS) turns the concept of middle-out abstraction into an operational principle, i.e. reducing complexity by collectively subsuming pattern-exhibiting model agents into according meta-agents, and reverting the process on demand. SOMACS enables the integration of required core components such as pattern detection and meta-agent management. It integrates with an established base platform framework for creating scalable self-organising multi-agent system simulations. Hence, it lays the foundation for the development and integration of these components and, ultimately, the universal deployment across diverse use cases. We outline its theoretical and practical underpinnings, provide a conceptual overview, discuss the achievements of a preliminary proof-of-concept implementation, and evaluate its potential with respect to performance, fidelity, and explainability. Finally, we outline its potential future extension into a broadly applicable middle-out abstraction framework.
