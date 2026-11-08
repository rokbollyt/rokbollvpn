.class public abstract La38;
.super Ljava/lang/Object;
.source "r8-map-id-2710c95bf169cfc7c8de6e5fe3d5484eac94ab79147e93b5c6da46338262cadc"


# direct methods
.method public static final a(Lbo8;Lnw4;Lip2;I)V
    .locals 0

    return-void
.end method

.method public static final b(JIII)J
    .locals 2

    .line 1
    invoke-static {p0, p1}, Lh78;->d(J)I

    .line 2
    .line 3
    .line 4
    move-result v0

    .line 5
    invoke-static {p0, p1}, Lh78;->c(J)I

    .line 6
    .line 7
    .line 8
    move-result v1

    .line 9
    if-ge v1, p2, :cond_0

    .line 10
    .line 11
    return-wide p0

    .line 12
    :cond_0
    if-gt v0, p2, :cond_2

    .line 13
    .line 14
    if-gt p3, v1, :cond_2

    .line 15
    .line 16
    sub-int/2addr p3, p2

    .line 17
    sub-int/2addr p4, p3

    .line 18
    if-ne v0, v1, :cond_1

    .line 19
    .line 20
    goto :goto_1

    .line 21
    :cond_1
    :goto_0
    add-int p2, v1, p4

    .line 22
    .line 23
    goto :goto_2

    .line 24
    :cond_2
    if-le v0, p2, :cond_3

    .line 25
    .line 26
    if-ge v1, p3, :cond_3

    .line 27
    .line 28
    add-int/2addr p2, p4

    .line 29
    move v0, p2

    .line 30
    goto :goto_2

    .line 31
    :cond_3
    if-lt v0, p3, :cond_4

    .line 32
    .line 33
    sub-int/2addr p3, p2

    .line 34
    sub-int/2addr p4, p3

    .line 35
    :goto_1
    add-int/2addr v0, p4

    .line 36
    goto :goto_0

    .line 37
    :cond_4
    if-ge p2, v0, :cond_5

    .line 38
    .line 39
    add-int v0, p2, p4

    .line 40
    .line 41
    sub-int/2addr p3, p2

    .line 42
    sub-int/2addr p4, p3

    .line 43
    add-int p2, p4, v1

    .line 44
    .line 45
    :cond_5
    :goto_2
    invoke-static {v0, p2}, Ltr7;->c(II)J

    .line 46
    .line 47
    .line 48
    move-result-wide p0

    .line 49
    return-wide p0
.end method

.method public static final c(Lxp5;Ljava/lang/String;Lr51;)Ljava/lang/Object;
    .locals 2

    .line 1
    new-instance v0, Lv38;

    .line 2
    .line 3
    const/4 v1, 0x6

    .line 4
    invoke-direct {v0, v1}, Lv38;-><init>(B)V

    .line 5
    .line 6
    .line 7
    invoke-interface {p0, p1, v0, p2}, Lxp5;->d(Ljava/lang/String;Ldn2;Lr51;)Ljava/lang/Object;

    .line 8
    .line 9
    .line 10
    move-result-object p0

    .line 11
    sget-object p1, Lb71;->X:Lb71;

    .line 12
    .line 13
    if-ne p0, p1, :cond_0

    .line 14
    .line 15
    return-object p0

    .line 16
    :cond_0
    sget-object p0, Lfm8;->a:Lfm8;

    .line 17
    .line 18
    return-object p0
.end method

.method public static d(Landroid/view/View;)Lwz;
    .locals 2

    .line 1
    sget v0, Landroid/os/Build$VERSION;->SDK_INT:I

    .line 2
    .line 3
    const/16 v1, 0x1a

    .line 4
    .line 5
    if-lt v0, v1, :cond_0

    .line 6
    .line 7
    invoke-static {p0}, Lze3;->t(Landroid/view/View;)Landroid/view/autofill/AutofillId;

    .line 8
    .line 9
    .line 10
    move-result-object p0

    .line 11
    new-instance v0, Lwz;

    .line 12
    .line 13
    invoke-direct {v0, p0}, Lwz;-><init>(Landroid/view/autofill/AutofillId;)V

    .line 14
    .line 15
    .line 16
    return-object v0

    .line 17
    :cond_0
    const/4 p0, 0x0

    .line 18
    return-object p0
.end method

.method public static final e(Lz28;)V
    .locals 4

    .line 1
    iget-object v0, p0, Lz28;->Z:Lph5;

    .line 2
    .line 3
    invoke-virtual {v0}, Lph5;->length()I

    .line 4
    .line 5
    .line 6
    move-result v1

    .line 7
    invoke-virtual {v0}, Lph5;->length()I

    .line 8
    .line 9
    .line 10
    move-result v0

    .line 11
    add-int/lit8 v0, v0, 0x1

    .line 12
    .line 13
    if-ltz v1, :cond_0

    .line 14
    .line 15
    if-ge v1, v0, :cond_0

    .line 16
    .line 17
    goto :goto_0

    .line 18
    :cond_0
    new-instance v2, Ljava/lang/StringBuilder;

    .line 19
    .line 20
    const-string v3, "Expected "

    .line 21
    .line 22
    invoke-direct {v2, v3}, Ljava/lang/StringBuilder;-><init>(Ljava/lang/String;)V

    .line 23
    .line 24
    .line 25
    invoke-virtual {v2, v1}, Ljava/lang/StringBuilder;->append(I)Ljava/lang/StringBuilder;

    .line 26
    .line 27
    .line 28
    const-string v3, " to be in [0, "

    .line 29
    .line 30
    invoke-virtual {v2, v3}, Ljava/lang/StringBuilder;->append(Ljava/lang/String;)Ljava/lang/StringBuilder;

    .line 31
    .line 32
    .line 33
    invoke-virtual {v2, v0}, Ljava/lang/StringBuilder;->append(I)Ljava/lang/StringBuilder;

    .line 34
    .line 35
    .line 36
    const-string v0, ")"

    .line 37
    .line 38
    invoke-virtual {v2, v0}, Ljava/lang/StringBuilder;->append(Ljava/lang/String;)Ljava/lang/StringBuilder;

    .line 39
    .line 40
    .line 41
    invoke-virtual {v2}, Ljava/lang/StringBuilder;->toString()Ljava/lang/String;

    .line 42
    .line 43
    .line 44
    move-result-object v0

    .line 45
    invoke-static {v0}, Led3;->a(Ljava/lang/String;)V

    .line 46
    .line 47
    .line 48
    :goto_0
    invoke-static {v1, v1}, Ltr7;->c(II)J

    .line 49
    .line 50
    .line 51
    move-result-wide v0

    .line 52
    iput-wide v0, p0, Lz28;->i0:J

    .line 53
    .line 54
    return-void
.end method

.method public static final f(Lz28;II)V
    .locals 2

    .line 1
    iget-object v0, p0, Lz28;->Z:Lph5;

    .line 2
    .line 3
    invoke-virtual {v0}, Lph5;->length()I

    .line 4
    .line 5
    .line 6
    move-result v0

    .line 7
    const/4 v1, 0x0

    .line 8
    invoke-static {p1, v1, v0}, Lkv3;->p(III)I

    .line 9
    .line 10
    .line 11
    move-result p1

    .line 12
    iget-object v0, p0, Lz28;->Z:Lph5;

    .line 13
    .line 14
    invoke-virtual {v0}, Lph5;->length()I

    .line 15
    .line 16
    .line 17
    move-result v0

    .line 18
    invoke-static {p2, v1, v0}, Lkv3;->p(III)I

    .line 19
    .line 20
    .line 21
    move-result p2

    .line 22
    invoke-static {p1, p2}, Ltr7;->c(II)J

    .line 23
    .line 24
    .line 25
    move-result-wide p1

    .line 26
    invoke-virtual {p0, p1, p2}, Lz28;->g(J)V

    .line 27
    .line 28
    .line 29
    return-void
.end method
