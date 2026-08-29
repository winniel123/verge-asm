---
title: Zone files
section: Getting started
order: 3
description: What a zone file is, why verge-asm wants one for removal detection, and how to export one from your DNS provider.
---

# Zone files

Uploading a zone file for a name scope enables **removal detection**. This guide
explains what a zone file is, why verge-asm wants one, and how to export one from the
place your DNS actually lives. It expands
[using.md → Upload a zone file](using.md#2-upload-a-zone-file), which is where the act
sits in the checklist.

---

## What a zone file is

A zone file is a plain-text listing of every DNS record published for a domain. It lists
the `A`, `AAAA`, `CNAME`, `MX`, `TXT` and delegation records in your zone. It is
the format authoritative name servers load, defined by RFC 1035's *master file*
syntax and universally called **BIND format**. One line per record:

```text
$ORIGIN example.com.
@         3600  IN  SOA   ns1.example.com. hostmaster.example.com. ( 2026081601 7200 3600 1209600 3600 )
@         3600  IN  NS    ns1.example.com.
www       3600  IN  A     203.0.113.10
api       3600  IN  A     203.0.113.11
mail      3600  IN  MX    10 mail.example.com.
```

The **apex** is the domain the file is *about* — the `$ORIGIN` / `SOA` name,
`example.com` above. verge-asm reads the file for the names your zone publishes. It
does not need a complete or signed file, only an honest snapshot of what you serve.

---

## Why supply one — removal detection

Without a zone file, verge-asm learns names only by **resolving** them and by what
other sources admit. That is enough to find what exists, but it cannot tell you a name
has *gone*. A name that stops resolving might be removed, might be a transient DNS
failure, or might simply be one your resolver stopped being asked about. Absence of an
answer is not evidence of removal.

A zone file closes that gap. It is your declaration of *the complete set of names I
publish*, dated to the moment you supplied it. When you upload a newer file that no
longer lists a name, that disappearance is a **positive fact you asserted**, not a
silence. So verge-asm can surface the removal instead of guessing. This is the one
place in v1 where two enumerable sources meet on the same timeline: your zone file
(`authority: declared`) against your own resolver's measured answers
(`authority: measured`). That meeting is what lets a disagreement be *reported* rather
than one source silently overwriting the other.

Because it carries this weight, the zone file is treated as a **supply act**, not a
mount:

- It is **uploaded, not mounted**. The upload instant is the observation instant, so
  *you stopped telling us* is detectable. Re-reading unchanged bytes on a schedule
  would produce a fresh observation of a stale fact. Supplying the file dates it to
  your act instead.
- A file whose **apex sits outside** the name scope it was uploaded against is
  **refused**, with the reason. Upload the `example.com` zone against the
  `example.com` scope.
- The zone is covered by its own **`zone` scan** on a **re-supply interval** you set
  (shipped **monthly**). This interval is your promise about how often you will
  re-export — not a schedule that re-reads the old file.
- Let a zone age past **two re-supply intervals** and it ages into a **`Gap`**: the
  system tells you the source went stale rather than trusting old bytes. Re-export and
  re-upload to close it.

---

## How to export a zone file

The exact menu path shifts as providers redesign their dashboards, but every
authoritative DNS host offers one of the routes below. You want BIND / RFC 1035
format — often labelled *"Export zone file"*, *"Export DNS records (BIND)"*, or just
*"Export"*.

### From a managed DNS provider (dashboard)

Most registrars and DNS hosts have a one-click export in the zone's DNS settings:

| Provider | Where to look |
| --- | --- |
| **Cloudflare** | DNS → Records → **Export** (downloads a BIND file). |
| **AWS Route 53** | No native BIND export in the console; use the CLI route below, or the community `cli53 export example.com` tool. |
| **Google Cloud DNS** | `gcloud dns record-sets export zonefile.txt --zone=<zone>`. |
| **Namecheap / GoDaddy / Porkbun** | Advanced DNS → **Export zone file**. |
| **Azure DNS** | `az network dns zone export -g <rg> -n example.com -f example.com.zone`. |

If your provider only shows records in a web table, the CLI or AXFR routes below give
you the same file without hand-transcribing rows.

### From the command line

`dig` pulls the whole zone in BIND format. This works if you run your own authoritative
name servers, or if your provider permits a zone transfer from an address you control:

```sh
dig AXFR example.com @ns1.example.com > example.com.zone
```

`AXFR` returns every record the server is authoritative for. Many providers disable
zone transfer by default — enable it for a trusted source address, or use the
provider's export button instead.

### From a self-hosted server

If you already run **BIND**, **Knot**, **NSD** or **PowerDNS**, the zone file (or its
export) *is* what those servers load. Copy the `db.example.com` / zone file straight
from your config, or dump it (`named-compilezone`, `pdnsutil`, `kzonecheck`
alongside your server's export command).

---

## Uploading it

Under **Scope → zone**, choose the name scope. Upload the file. verge-asm checks the
apex against the scope, records the supply instant, and schedules the `zone` scan on
your re-supply interval. Set the interval to match how often you actually re-export. If
you re-export monthly, leave it monthly. The two-interval `Gap` gives you a full extra
cycle of slack before staleness is called.
