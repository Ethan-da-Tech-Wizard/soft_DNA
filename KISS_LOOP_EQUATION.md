# The KISS Loop Equation

Formalizing the chunked-ingestion LLM scanning loop ("read 75%, summarize,
verify, repair, repeat") as mathematics. Guiding adage: **Keep It Simple,
Stupid.**

---

## 0. KISS itself, as an equation

Among all designs that satisfy the requirements, pick the least complex one:

```
design* = argmin K(d)   subject to   R(d) ⊇ R_required
          d ∈ D
```

where `K(d)` is the complexity of design `d` and `R(d)` is the set of
requirements it satisfies. That is the whole adage: *minimize complexity,
constrained by correctness — never the reverse.*

Applied here, it means the entire scanning architecture must reduce to
**one state vector, one update rule, one stop condition.** It does:

---

## 1. Definitions

| Symbol | Meaning |
|--------|---------|
| `X`    | The source (codebase), a token sequence of total length `T` |
| `C`    | The model's max context window, in tokens |
| `p_n`  | Read pointer: first unconsumed token at the start of turn `n` (`p_0 = 0`) |
| `Σ_n`  | Accumulated summary memory at the start of turn `n` (`Σ_0 = ∅`) |
| `ρ_n`  | Ingestion ratio for turn `n` — fraction of `C` spent reading |
| `B(x)` | Boundary snap: the largest *natural boundary* ≤ `x` (end of sentence, function, section, table, topic) |
| `f(Σ, chunk)` | The LLM: fold a chunk into the summary |
| `V(Σ)` | Validity check ∈ {0, 1}: did the summary end naturally, completely, coherently? |

The **state** of the whole system is just three values:

```
S_n = (Σ_n, p_n, ρ_n)
```

---

## 2. The update rule (one turn = one application of F)

Each turn reads a boundary-snapped chunk, folds it into memory, checks the
result, and either advances or holds position at half throttle:

```
q_n     = B( p_n + ρ_n·C )                     — candidate end, snapped to a clean break
Σ_{n+1} = f( Σ_n , X[p_n : q_n] )              — deep summary of what was safely read
v_n     = V( Σ_{n+1} )                          — summary completion check

p_{n+1} = v_n · q_n  +  (1 − v_n) · p_n        — advance only if the summary is valid
ρ_{n+1} = 1/2 + 1/4 · v_n                       — 75% when healthy, 50% after a failure
```

That last line **is** your entire repair mode: `ρ = 0.75` while summaries
are clean, drop to `ρ = 0.50` the moment one isn't, and because
`p_{n+1} = p_n` on failure, the retry automatically "starts at the
beginning of the idea that was cut off" — the pointer never moved past it.

Compactly, the whole system is a fixed-point iteration:

```
S_{n+1} = F(S_n)
```

---

## 3. The stop condition

```
HALT  when  p_n ≥ T  and  V(Σ_n) = 1
```

Everything has been read, and the final summary passed its own completion
check (your FINAL COMPLETION CHECK box). Until both hold, loop — all day if
it must.

---

## 4. The invariant (why there are "no flaws")

The loop maintains one invariant, checkable every turn:

```
I(n):  every token in X[0 : p_n] is represented in Σ_n,  and  V(Σ_n) = 1
```

- `I(0)` holds trivially (nothing read, empty summary).
- `F` preserves it: the pointer only advances (`p_{n+1} = q_n`) in the same
  step where the new summary passed `V`. A failed summary freezes `p`.
- Therefore at HALT, `Σ` validly represents all of `X`. ∎

Coverage is monotone: `p_0 ≤ p_1 ≤ p_2 ≤ …` — the system can stall to
repair, but it can never lose ground and never skip.

**The one real flaw the math exposes:** if a single atomic unit (one giant
function, one huge table) is longer than `ρ·C`, then `B(p + ρC) = p`, the
chunk is empty, and the loop spins forever. Guard required:

```
if q_n = p_n:   q_n = p_n + ρ_n·C     — hard split; summarize with an explicit
                                        "continues past cutoff" marker in Σ
```

That guard is the only special case the architecture needs. Resist adding
others (KISS).

---

## 5. Lightweight real-time diagnostics

Four numbers per turn — printable on one line, graphable, greppable:

```
turn n:   γ_n = p_n / T        coverage        (must be non-decreasing → 1.0)
          ρ_n ∈ {0.75, 0.50}   throttle        (0.50 = currently repairing)
          v_n ∈ {1, 0}         summary health  (streaks of 0 = trouble spot)
          n − argmax(p)        stall count     (turns since p last advanced;
                                                > 2 ⇒ atomic-unit guard firing)
```

Health of the entire run reduces to: **γ is climbing, and stall count is
small.** If both hold, the loop is provably converging; if either breaks,
the diagnostic line tells you exactly which box of the flowchart is stuck.

---

## 6. The whole architecture in three lines

```
State:   S = (Σ, p, ρ)
Step:    Σ ← f(Σ, X[p : B(p+ρC)]);   v ← V(Σ);   p ← v·B(p+ρC) + (1−v)·p;   ρ ← ½ + ¼v
Stop:    p ≥ T  ∧  v = 1
```

Keep it simple, stupid.
