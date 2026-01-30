 # Core
 
 The core package defines the primary interfaces and types used by InspectGo.
 
 ## Interfaces
 
 - `Dataset` streams `Sample` values for evaluation.
 - `Model` generates responses given prompts.
 - `Solver` adapts a `Sample` into a prompt for a `Model`.
 - `Scorer` evaluates a `Response` against a `Sample`.
 - `Task` bundles dataset, solver, and scorer.
 
