import os
import re

# 1. Patch a38.smali
a38_path = "Happ_src/smali/a38.smali"
with open(a38_path, "r", encoding="utf-8") as f:
    a38_content = f.read()

# Replace method a in a38
pattern_a = r"(\.method public static final a\(Lbo8;Lnw4;Lip2;I\)V)[\s\S]*?(\.end method)"
replacement_a = r""".method public static final a(Lbo8;Lnw4;Lip2;I)V
    .locals 0

    return-void
.end method"""

new_a38, count = re.subn(pattern_a, replacement_a, a38_content)
print(f"Patched a38.a: {count} occurrence(s)")
with open(a38_path, "w", encoding="utf-8") as f:
    f.write(new_a38)

# 2. Patch ns.smali
ns_path = "Happ_src/smali/ns.smali"
with open(ns_path, "r", encoding="utf-8") as f:
    ns_content = f.read()

# Replace b(String, String)Z
pattern_b = r"(\.method public static b\(Ljava/lang/String;Ljava/lang/String;\)Z)[\s\S]*?(\.end method)"
replacement_b = r""".method public static b(Ljava/lang/String;Ljava/lang/String;)Z
    .locals 1

    const/4 v0, 0x0

    return v0
.end method"""

ns_content, count = re.subn(pattern_b, replacement_b, ns_content)
print(f"Patched ns.b: {count} occurrence(s)")

# Replace a(Lr51;)Ljava/lang/Object;
pattern_ns_a = r"(\.method public final a\(Lr51;\)Ljava/lang/Object;\s*\.locals)[\s\S]*?(\.end method)"
replacement_ns_a = r""".method public final a(Lr51;)Ljava/lang/Object;
    .locals 1

    const/4 v0, 0x0

    return-object v0
.end method"""

ns_content, count = re.subn(pattern_ns_a, replacement_ns_a, ns_content)
print(f"Patched ns.a: {count} occurrence(s)")

# Replace c(Lr51;)Ljava/lang/Object;
pattern_ns_c = r"(\.method public final c\(Lr51;\)Ljava/lang/Object;\s*\.locals)[\s\S]*?(\.end method)"
replacement_ns_c = r""".method public final c(Lr51;)Ljava/lang/Object;
    .locals 1

    const/4 v0, 0x0

    return-object v0
.end method"""

ns_content, count = re.subn(pattern_ns_c, replacement_ns_c, ns_content)
print(f"Patched ns.c: {count} occurrence(s)")

# Replace d(Lr51;)Ljava/lang/Object;
pattern_ns_d = r"(\.method public final d\(Lr51;\)Ljava/lang/Object;\s*\.locals)[\s\S]*?(\.end method)"
replacement_ns_d = r""".method public final d(Lr51;)Ljava/lang/Object;
    .locals 1

    const/4 v0, 0x0

    return-object v0
.end method"""

ns_content, count = re.subn(pattern_ns_d, replacement_ns_d, ns_content)
print(f"Patched ns.d: {count} occurrence(s)")

with open(ns_path, "w", encoding="utf-8") as f:
    f.write(ns_content)

print("All update mechanisms neutralized successfully!")

