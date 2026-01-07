# SRE Simulation Game: 100 - Go To Space

This directory contains the interactive SRE simulation game **"100 - Go To Space"**.

## Purpose
This game is an educational tool designed to illustrate key Site Reliability Engineering (SRE) concepts featured in this repository:
- **SLOs & SLIs**: Maintaining "nines" of uptime.
- **Error Budgets**: Managing risk and deployment velocity.
- **Toil Reduction**: Automating manual tasks through "Infrastructure" upgrades.
- **Incident Response**: Mitigating random failure events.

## Why is this in its own directory?
The game is a self-contained HTML/Javascript application. It is stored here to keep it separate from the core backend logic (Go) and infrastructure (Terraform/Kubernetes) that define the "Production Journey" this repository demonstrates.

The game is deployed to GitHub Pages at:
[https://stevemcghee.github.io/go-to-production/game/](https://stevemcghee.github.io/go-to-production/game/)
