# The On-Disk Node Layout

> How a B-tree node is encoded into a single fixed-size page of bytes, and why that
> layout is shaped the way it is.
>
> **DDIA link:** Chapter 3 — "B-Trees". Pages, fixed-size blocks, and references to
> data by *location on disk* instead of by memory pointer.

---

## 1. Recap: a node is a box

In Step 0 a node was just a conceptual box holding a short **sorted** list of keys (and,
for leaves, their values). That mental model is still correct. What changes now is *where*
the box lives: no longer in RAM as a struct full of pointers, but on **disk** as a flat
run of bytes.

On disk, each box is exactly one **page** — a fixed chunk of **4096 bytes**. Every page is
the same size, which gives us one powerful property:

> The database file is a row of equal-sized boxes, so **page number `n` starts at byte
> `n × 4096`.**

To locate a page you *multiply*. You never scan the file looking for it. This is the first
and most important reason the size is fixed: **random access by arithmetic.**

Why 4096 specifically? It matches the unit the operating system and SSD hardware already
read and write in (the OS page size and the typical flash block). Reading 4096 bytes costs
the same as reading 1 byte, because the hardware fetches the whole block anyway. Aligning
our node to that block means one node read = one hardware read. No waste, no straddling two
blocks.

---

## 2. The problem: variable-length data

Keys and values do **not** have a fixed length. `"cat"` is 3 bytes; an email address might
be 30. If we naively pack them end to end:

```
catdog942
```

we are stuck. Where does `cat` end and `dog` begin? Is the value `9`, `94`, or `942`? The
bytes carry no boundaries.

The obvious patch — **write a length before each item** — fixes the boundary problem but
not the *access* problem:

```
[3]cat[3]dog[3]942      (lengths shown in brackets)
```

Now you can parse the items in order, but to reach **item 5** you must walk past items 0, 1,
2, 3, 4, reading each length and skipping forward. That is **O(n) per lookup inside a single
node.** Step 3's search algorithm wants to **jump straight to any item** — ideally O(1) or
O(log n) — so sequential walking is too slow.

We need two things at once:
1. Variable-length data (keys/values differ in size).
2. Constant-time access to the *i*-th item.

These seem to conflict. The resolution is a classic trick.

---

## 3. The solution: a table of contents

Think of a physical book. Chapters have wildly different lengths, yet you can open
**chapter 7 instantly** — because the front of the book has a **fixed-size table of
contents** listing where each chapter starts. The table entries are all the same size (a
page number), so finding "the entry for chapter 7" is pure arithmetic, and that entry then
points you to the variable-length content.

A node copies this exact design:

- The **front** of the node holds a set of small **fixed-size slots** (the table of
  contents).
- The **variable-size data** comes *after* those slots.

Because the slots are fixed-size, slot *i* lives at a position you can compute directly.
The slot tells you where item *i*'s data is. Two steps, both O(1): **compute slot position
→ read the pointer/offset it holds → jump to the data.**

---

## 4. The full layout

```
┌────────┬────────┬─────────────────┬─────────────────┬───────────────────┬─────────┐
│ btype  │ nkeys  │    pointers     │    offsets      │  key-value pairs  │ unused  │
│ 2 B    │ 2 B    │ nkeys × 8 B     │ nkeys × 2 B     │ variable size     │ zeros   │
└────────┴────────┴─────────────────┴─────────────────┴───────────────────┴─────────┘
  └─ header (4 B) ─┘   └──── fixed-size slots: easy to jump to ────┘   the actual data
```

| Region     | Holds                                               | Why it's needed |
|------------|-----------------------------------------------------|-----------------|
| `btype`    | `1` = internal node, `2` = leaf                     | Code must know which kind of box it's reading before it can interpret the rest. |
| `nkeys`    | Number of keys in this node                         | The sizes of the next two regions (`pointers`, `offsets`) are *derived* from it. You cannot parse the node without knowing `nkeys` first. |
| `pointers` | One child **page number** per key (8 B each)        | Internal nodes: "keys from here onward live in page *N*." Leaves: unused, left as zeros. Keeping one shared format for both node kinds keeps the code simple. |
| `offsets`  | Where each pair **ends** (2 B each)                 | This is the table of contents — the fixed-size slots that make jumping possible. |
| `pairs`    | Each pair is `[klen 2B][vlen 2B][key][value]`       | The real data. Each pair records **its own** key length and value length, so it is self-describing. |
| `unused`   | Zeros                                               | Padding out to 4096. Free space the node can grow into before it must split. |

### Why the header must come first, in this order

Parsing is strictly sequential at the start:

1. Read `btype` (fixed position, bytes 0–1) → now you know how to interpret pointers.
2. Read `nkeys` (fixed position, bytes 2–3) → now you know how *long* the `pointers` and
   `offsets` regions are, so you know where every later region begins.

Everything downstream depends on `nkeys`. That is why it is nailed to a fixed location at
the very front. It is the key that unlocks the variable geometry of the rest of the node.

---

## 5. How offsets work

This is the subtle part, so go slowly.

- Offsets are counted **from the start of the `pairs` region**, not from the start of the
  page. (The pairs region itself begins at a position you compute from `nkeys` — see the
  worked example.)
- **Pair 0 always starts at offset 0.** Since that is always true, it carries no
  information, so **it is not stored.** The `offsets` array stores the *end* of each pair.
- **Pair *i* starts where pair *i−1* ended.** So `offset[i]` (the end of pair *i*) is also
  the *start* of pair *i+1*.
- The **last stored offset** is the end of the last pair, which equals the **total size of
  all pairs combined.** Step 6 uses exactly this number to decide whether the node has
  grown too big and must split.

Storing *ends* rather than *starts* is what lets a single array give you both the start and
the end of any pair:

```
start of pair i = offset[i-1]   (0 if i == 0)
end   of pair i = offset[i]
length of pair i = offset[i] - offset[i-1]
```

One fixed-size array, constant-time access to any pair's boundaries. That is the whole
payoff of the table-of-contents design.

---

## 6. Worked example

A **leaf** node holding two pairs: `"cat" : "9"` and `"dog" : "42"`.

### Step A — size each pair

Each pair is `[klen 2B][vlen 2B][key bytes][value bytes]`.

```
pair 0 ("cat":"9")  = 2 (klen) + 2 (vlen) + 3 ("cat") + 1 ("9")  = 8 bytes  → ends at 8
pair 1 ("dog":"42") = 2        + 2        + 3 ("dog") + 2 ("42")  = 9 bytes  → ends at 17
```

So the stored offsets are `[8, 17]`. (17 is `0x11` in hex.)

### Step B — find where the pairs region starts

Walk the layout from the front, using `nkeys = 2`:

```
header   = 4 bytes            (btype 2B + nkeys 2B)
pointers = nkeys × 8 = 2 × 8  = 16 bytes
offsets  = nkeys × 2 = 2 × 2  =  4 bytes
                               ───────────
pairs region begins at byte     24
```

### Step C — the raw bytes

```
00000000  02 00 02 00 00 00 00 00  00 00 00 00 00 00 00 00  |................|
00000010  00 00 00 00 08 00 11 00  03 00 01 00 63 61 74 39  |............cat9|
00000020  03 00 02 00 64 6f 67 34  32                       |....dog42|
```

### Step D — decode it byte by byte

| Bytes  | Raw                        | Meaning |
|--------|----------------------------|---------|
| 0–1    | `02 00`                    | `btype = 2` (leaf) |
| 2–3    | `02 00`                    | `nkeys = 2` |
| 4–19   | zeros                       | 2 pointers × 8 B — **unused in a leaf**, all zero |
| 20–21  | `08 00`                    | `offset[0]` = pair 0 **ends** at 8 |
| 22–23  | `11 00`                    | `offset[1]` = pair 1 **ends** at 17 (`0x11`) |
| 24–31  | `03 00 01 00 63 61 74 39`  | pair 0: `klen=3`, `vlen=1`, `63 61 74`=`"cat"`, `39`=`"9"` |
| 32–40  | `03 00 02 00 64 6f 67 34 32` | pair 1: `klen=3`, `vlen=2`, `64 6f 67`=`"dog"`, `34 32`=`"42"` |

Everything from byte 41 to byte 4095 is zero — the `unused` region, the slack this node can
fill before it must split.

### Notes on reading the hex

- All multi-byte integers are **little-endian**, so `08 00` reads as `8`, and `11 00` reads
  as `0x11 = 17`. The low byte comes first. (This is why the earlier code used
  `binary.LittleEndian`.)
- The ASCII column on the right (`|...cat9|`) shows which bytes happen to be printable
  characters — a quick sanity check that your keys and values landed where you expect.
- `63 61 74` → `c a t`; `39` → `9`; `64 6f 67` → `d o g`; `34 32` → `4 2`. These are plain
  ASCII codes.

---

## 7. Why this design, in one paragraph

A node must store **variable-length** keys and values yet still allow **constant-time jumps**
to any one of them, all inside a **fixed 4096-byte** page so that page *n* can be found by
multiplication. The answer is a book's table of contents: a small, fixed-size header
(`btype`, `nkeys`) that tells you the node's shape, followed by fixed-size slots
(`pointers`, `offsets`) that you can index with arithmetic, followed by the variable-length
`pairs`. Offsets store each pair's *end*, which simultaneously gives you the next pair's
*start* and, in the final slot, the total data size used for split decisions. Internal nodes
and leaves share one format — leaves simply leave the `pointers` region zeroed — so a single
set of encode/decode routines serves both.
