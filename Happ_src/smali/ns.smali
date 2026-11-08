.class public final Lns;
.super Ljava/lang/Object;
.source "r8-map-id-2710c95bf169cfc7c8de6e5fe3d5484eac94ab79147e93b5c6da46338262cadc"


# instance fields
.field public final a:Lvn5;

.field public final b:Lto8;

.field public final c:Ll51;

.field public final d:Lis;

.field public volatile e:J

.field public final f:Ljava/lang/String;

.field public final g:Lue2;


# direct methods
.method public constructor <init>()V
    .locals 4

    .line 1
    new-instance v0, Lvn5;

    .line 2
    .line 3
    const/16 v1, 0x1c

    .line 4
    .line 5
    invoke-direct {v0, v1}, Lvn5;-><init>(B)V

    .line 6
    .line 7
    .line 8
    new-instance v1, Lto8;

    .line 9
    .line 10
    invoke-direct {v1}, Lto8;-><init>()V

    .line 11
    .line 12
    .line 13
    sget-object v2, Lsu/happ/proxyutility/HappApplication;->M0:Lsu/happ/proxyutility/HappApplication;

    .line 14
    .line 15
    invoke-static {}, Lh32;->h()Lsu/happ/proxyutility/HappApplication;

    .line 16
    .line 17
    .line 18
    move-result-object v2

    .line 19
    iget-object v2, v2, Lsu/happ/proxyutility/HappApplication;->Y:Lny7;

    .line 20
    .line 21
    invoke-virtual {v2}, Lny7;->getValue()Ljava/lang/Object;

    .line 22
    .line 23
    .line 24
    move-result-object v2

    .line 25
    check-cast v2, Lxo8;

    .line 26
    .line 27
    invoke-virtual {v2}, Ljava/lang/Object;->getClass()Ljava/lang/Class;

    .line 28
    .line 29
    .line 30
    invoke-direct {p0}, Ljava/lang/Object;-><init>()V

    .line 31
    .line 32
    .line 33
    iput-object v0, p0, Lns;->a:Lvn5;

    .line 34
    .line 35
    iput-object v1, p0, Lns;->b:Lto8;

    .line 36
    .line 37
    invoke-static {}, Lgv7;->b()Lfv7;

    .line 38
    .line 39
    .line 40
    move-result-object v0

    .line 41
    sget-object v2, Lrp1;->a:Lmf1;

    .line 42
    .line 43
    sget-object v2, Lxe1;->Z:Lxe1;

    .line 44
    .line 45
    invoke-static {v0, v2}, Ly51;->Z(Lp61;Lr61;)Lr61;

    .line 46
    .line 47
    .line 48
    move-result-object v0

    .line 49
    invoke-static {v0}, Lih3;->b(Lr61;)Ll51;

    .line 50
    .line 51
    .line 52
    move-result-object v0

    .line 53
    iput-object v0, p0, Lns;->c:Ll51;

    .line 54
    .line 55
    new-instance v0, Lis;

    .line 56
    .line 57
    invoke-direct {v0, p0}, Lis;-><init>(Lns;)V

    .line 58
    .line 59
    .line 60
    iput-object v0, p0, Lns;->d:Lis;

    .line 61
    .line 62
    const-wide/16 v2, -0x1

    .line 63
    .line 64
    iput-wide v2, p0, Lns;->e:J

    .line 65
    .line 66
    iget-object v0, v1, Lto8;->e:Ljava/lang/String;

    .line 67
    .line 68
    iput-object v0, p0, Lns;->f:Ljava/lang/String;

    .line 69
    .line 70
    iget-object v0, v1, Lto8;->g:Lue2;

    .line 71
    .line 72
    iput-object v0, p0, Lns;->g:Lue2;

    .line 73
    .line 74
    return-void
.end method

.method public static b(Ljava/lang/String;Ljava/lang/String;)Z
    .locals 1

    const/4 v0, 0x0

    return v0
.end method

.method public static f(Ljava/lang/String;)Ljava/util/List;
    .locals 4

    .line 1
    sget-object v0, Los;->a:Lnb6;

    .line 2
    .line 3
    invoke-virtual {v0}, Ljava/lang/Object;->getClass()Ljava/lang/Class;

    .line 4
    .line 5
    .line 6
    invoke-virtual {p0}, Ljava/lang/Object;->getClass()Ljava/lang/Class;

    .line 7
    .line 8
    .line 9
    const/4 v1, 0x0

    .line 10
    invoke-static {v1}, Lgl7;->O0(I)V

    .line 11
    .line 12
    .line 13
    iget-object v0, v0, Lnb6;->X:Ljava/util/regex/Pattern;

    .line 14
    .line 15
    invoke-virtual {v0, p0}, Ljava/util/regex/Pattern;->matcher(Ljava/lang/CharSequence;)Ljava/util/regex/Matcher;

    .line 16
    .line 17
    .line 18
    move-result-object v0

    .line 19
    invoke-virtual {v0}, Ljava/util/regex/Matcher;->find()Z

    .line 20
    .line 21
    .line 22
    move-result v2

    .line 23
    if-nez v2, :cond_0

    .line 24
    .line 25
    invoke-virtual {p0}, Ljava/lang/Object;->toString()Ljava/lang/String;

    .line 26
    .line 27
    .line 28
    move-result-object p0

    .line 29
    invoke-static {p0}, Lv70;->C(Ljava/lang/Object;)Ljava/util/List;

    .line 30
    .line 31
    .line 32
    move-result-object p0

    .line 33
    goto :goto_0

    .line 34
    :cond_0
    new-instance v2, Ljava/util/ArrayList;

    .line 35
    .line 36
    const/16 v3, 0xa

    .line 37
    .line 38
    invoke-direct {v2, v3}, Ljava/util/ArrayList;-><init>(I)V

    .line 39
    .line 40
    .line 41
    :cond_1
    invoke-virtual {v0}, Ljava/util/regex/Matcher;->start()I

    .line 42
    .line 43
    .line 44
    move-result v3

    .line 45
    invoke-virtual {p0, v1, v3}, Ljava/lang/String;->subSequence(II)Ljava/lang/CharSequence;

    .line 46
    .line 47
    .line 48
    move-result-object v1

    .line 49
    invoke-virtual {v1}, Ljava/lang/Object;->toString()Ljava/lang/String;

    .line 50
    .line 51
    .line 52
    move-result-object v1

    .line 53
    invoke-virtual {v2, v1}, Ljava/util/ArrayList;->add(Ljava/lang/Object;)Z

    .line 54
    .line 55
    .line 56
    invoke-virtual {v0}, Ljava/util/regex/Matcher;->end()I

    .line 57
    .line 58
    .line 59
    move-result v1

    .line 60
    invoke-virtual {v0}, Ljava/util/regex/Matcher;->find()Z

    .line 61
    .line 62
    .line 63
    move-result v3

    .line 64
    if-nez v3, :cond_1

    .line 65
    .line 66
    invoke-virtual {p0}, Ljava/lang/String;->length()I

    .line 67
    .line 68
    .line 69
    move-result v0

    .line 70
    invoke-virtual {p0, v1, v0}, Ljava/lang/String;->subSequence(II)Ljava/lang/CharSequence;

    .line 71
    .line 72
    .line 73
    move-result-object p0

    .line 74
    invoke-virtual {p0}, Ljava/lang/Object;->toString()Ljava/lang/String;

    .line 75
    .line 76
    .line 77
    move-result-object p0

    .line 78
    invoke-virtual {v2, p0}, Ljava/util/ArrayList;->add(Ljava/lang/Object;)Z

    .line 79
    .line 80
    .line 81
    move-object p0, v2

    .line 82
    :goto_0
    invoke-interface {p0}, Ljava/util/List;->isEmpty()Z

    .line 83
    .line 84
    .line 85
    move-result v0

    .line 86
    if-nez v0, :cond_3

    .line 87
    .line 88
    invoke-interface {p0}, Ljava/util/List;->size()I

    .line 89
    .line 90
    .line 91
    move-result v0

    .line 92
    invoke-interface {p0, v0}, Ljava/util/List;->listIterator(I)Ljava/util/ListIterator;

    .line 93
    .line 94
    .line 95
    move-result-object v0

    .line 96
    :goto_1
    invoke-interface {v0}, Ljava/util/ListIterator;->hasPrevious()Z

    .line 97
    .line 98
    .line 99
    move-result v1

    .line 100
    if-eqz v1, :cond_3

    .line 101
    .line 102
    invoke-interface {v0}, Ljava/util/ListIterator;->previous()Ljava/lang/Object;

    .line 103
    .line 104
    .line 105
    move-result-object v1

    .line 106
    check-cast v1, Ljava/lang/String;

    .line 107
    .line 108
    invoke-virtual {v1}, Ljava/lang/String;->length()I

    .line 109
    .line 110
    .line 111
    move-result v1

    .line 112
    if-nez v1, :cond_2

    .line 113
    .line 114
    goto :goto_1

    .line 115
    :cond_2
    invoke-interface {v0}, Ljava/util/ListIterator;->nextIndex()I

    .line 116
    .line 117
    .line 118
    move-result v0

    .line 119
    add-int/lit8 v0, v0, 0x1

    .line 120
    .line 121
    invoke-static {p0, v0}, Lvv0;->Q0(Ljava/lang/Iterable;I)Ljava/util/List;

    .line 122
    .line 123
    .line 124
    move-result-object p0

    .line 125
    return-object p0

    .line 126
    :cond_3
    sget-object p0, Lb02;->X:Lb02;

    .line 127
    .line 128
    return-object p0
.end method


# virtual methods
.method public final a(Lr51;)Ljava/lang/Object;
    .locals 1

    const/4 v0, 0x0

    return-object v0
.end method

.method public final c(Lr51;)Ljava/lang/Object;
    .locals 1

    const/4 v0, 0x0

    return-object v0
.end method

.method public final d(Lr51;)Ljava/lang/Object;
    .locals 1

    const/4 v0, 0x0

    return-object v0
.end method

.method public final e(Lbo8;Lr51;)Ljava/lang/Object;
    .locals 11

    .line 1
    instance-of v0, p2, Lms;

    .line 2
    .line 3
    if-eqz v0, :cond_0

    .line 4
    .line 5
    move-object v0, p2

    .line 6
    check-cast v0, Lms;

    .line 7
    .line 8
    iget v1, v0, Lms;->o0:I

    .line 9
    .line 10
    const/high16 v2, -0x80000000

    .line 11
    .line 12
    and-int v3, v1, v2

    .line 13
    .line 14
    if-eqz v3, :cond_0

    .line 15
    .line 16
    sub-int/2addr v1, v2

    .line 17
    iput v1, v0, Lms;->o0:I

    .line 18
    .line 19
    goto :goto_0

    .line 20
    :cond_0
    new-instance v0, Lms;

    .line 21
    .line 22
    invoke-direct {v0, p0, p2}, Lms;-><init>(Lns;Lr51;)V

    .line 23
    .line 24
    .line 25
    :goto_0
    iget-object p2, v0, Lms;->m0:Ljava/lang/Object;

    .line 26
    .line 27
    sget-object v1, Lb71;->X:Lb71;

    .line 28
    .line 29
    iget v2, v0, Lms;->o0:I

    .line 30
    .line 31
    const/4 v3, 0x0

    .line 32
    const/4 v4, 0x0

    .line 33
    const/4 v5, 0x1

    .line 34
    const/4 v6, 0x2

    .line 35
    if-eqz v2, :cond_3

    .line 36
    .line 37
    if-eq v2, v5, :cond_2

    .line 38
    .line 39
    if-ne v2, v6, :cond_1

    .line 40
    .line 41
    iget p1, v0, Lms;->l0:I

    .line 42
    .line 43
    iget-object v1, v0, Lms;->k0:Lns;

    .line 44
    .line 45
    iget-object v2, v0, Lms;->j0:Lsu/happ/proxyutility/service/d;

    .line 46
    .line 47
    iget-object v3, v0, Lms;->i0:Lsu/happ/proxyutility/HappApplication;

    .line 48
    .line 49
    iget-object v5, v0, Lms;->h0:Ljava/lang/String;

    .line 50
    .line 51
    iget-object v0, v0, Lms;->g0:Ljava/lang/String;

    .line 52
    .line 53
    :try_start_0
    invoke-static {p2}, Lw05;->W(Ljava/lang/Object;)V
    :try_end_0
    .catchall {:try_start_0 .. :try_end_0} :catchall_0

    .line 54
    .line 55
    .line 56
    move-object v7, v0

    .line 57
    move-object v6, v5

    .line 58
    move-object v5, v3

    .line 59
    goto :goto_3

    .line 60
    :catchall_0
    move-exception v0

    .line 61
    move-object p0, v0

    .line 62
    goto/16 :goto_5

    .line 63
    .line 64
    :cond_1
    const-string p0, "call to \'resume\' before \'invoke\' with coroutine"

    .line 65
    .line 66
    invoke-static {p0}, Lct;->j(Ljava/lang/String;)V

    .line 67
    .line 68
    .line 69
    return-object v3

    .line 70
    :cond_2
    iget p1, v0, Lms;->l0:I

    .line 71
    .line 72
    iget-object v2, v0, Lms;->f0:Lbo8;

    .line 73
    .line 74
    :try_start_1
    invoke-static {p2}, Lw05;->W(Ljava/lang/Object;)V
    :try_end_1
    .catchall {:try_start_1 .. :try_end_1} :catchall_0

    .line 75
    .line 76
    .line 77
    move p2, p1

    .line 78
    move-object p1, v2

    .line 79
    goto :goto_1

    .line 80
    :cond_3
    invoke-static {p2}, Lw05;->W(Ljava/lang/Object;)V

    .line 81
    .line 82
    .line 83
    :try_start_2
    iget-object p2, p0, Lns;->b:Lto8;

    .line 84
    .line 85
    iput-object p1, v0, Lms;->f0:Lbo8;

    .line 86
    .line 87
    iput v4, v0, Lms;->l0:I

    .line 88
    .line 89
    iput v5, v0, Lms;->o0:I

    .line 90
    .line 91
    invoke-virtual {p2, p1, v0}, Lto8;->f(Lbo8;Lr51;)Ljava/lang/Object;

    .line 92
    .line 93
    .line 94
    move-result-object p2
    :try_end_2
    .catchall {:try_start_2 .. :try_end_2} :catchall_2

    .line 95
    if-ne p2, v1, :cond_4

    .line 96
    .line 97
    goto :goto_2

    .line 98
    :cond_4
    move p2, v4

    .line 99
    :goto_1
    :try_start_3
    sget-object v2, Lsu/happ/proxyutility/service/d;->a:Lsu/happ/proxyutility/service/d;

    .line 100
    .line 101
    sget-object v5, Lsu/happ/proxyutility/HappApplication;->M0:Lsu/happ/proxyutility/HappApplication;

    .line 102
    .line 103
    invoke-static {}, Lh32;->h()Lsu/happ/proxyutility/HappApplication;

    .line 104
    .line 105
    .line 106
    move-result-object v5

    .line 107
    iget-object p1, p1, Lbo8;->Z:Ljava/lang/String;

    .line 108
    .line 109
    iget-object v7, p0, Lns;->b:Lto8;

    .line 110
    .line 111
    iget-object v7, v7, Lto8;->f:Landroid/net/Uri;

    .line 112
    .line 113
    invoke-virtual {v7}, Landroid/net/Uri;->getPath()Ljava/lang/String;

    .line 114
    .line 115
    .line 116
    move-result-object v7

    .line 117
    if-eqz v7, :cond_6

    .line 118
    .line 119
    iget-object v8, p0, Lns;->b:Lto8;

    .line 120
    .line 121
    iput-object v3, v0, Lms;->f0:Lbo8;

    .line 122
    .line 123
    iput-object v7, v0, Lms;->g0:Ljava/lang/String;

    .line 124
    .line 125
    iput-object p1, v0, Lms;->h0:Ljava/lang/String;

    .line 126
    .line 127
    iput-object v5, v0, Lms;->i0:Lsu/happ/proxyutility/HappApplication;

    .line 128
    .line 129
    iput-object v2, v0, Lms;->j0:Lsu/happ/proxyutility/service/d;

    .line 130
    .line 131
    iput-object p0, v0, Lms;->k0:Lns;

    .line 132
    .line 133
    iput p2, v0, Lms;->l0:I

    .line 134
    .line 135
    iput v6, v0, Lms;->o0:I

    .line 136
    .line 137
    iget-object v3, v8, Lto8;->h:Lo;

    .line 138
    .line 139
    invoke-static {v3, v0}, Lih3;->r(Lue2;Lr51;)Ljava/lang/Object;

    .line 140
    .line 141
    .line 142
    move-result-object v0
    :try_end_3
    .catchall {:try_start_3 .. :try_end_3} :catchall_1

    .line 143
    if-ne v0, v1, :cond_5

    .line 144
    .line 145
    :goto_2
    return-object v1

    .line 146
    :cond_5
    move-object v1, p0

    .line 147
    move-object v6, p1

    .line 148
    move p1, p2

    .line 149
    move-object p2, v0

    .line 150
    :goto_3
    :try_start_4
    check-cast p2, Ljava/lang/Number;

    .line 151
    .line 152
    invoke-virtual {p2}, Ljava/lang/Number;->longValue()J

    .line 153
    .line 154
    .line 155
    move-result-wide v8

    .line 156
    iget-object v10, p0, Lns;->d:Lis;

    .line 157
    .line 158
    invoke-virtual {v2}, Ljava/lang/Object;->getClass()Ljava/lang/Class;

    .line 159
    .line 160
    .line 161
    invoke-static/range {v5 .. v10}, Lsu/happ/proxyutility/service/d;->e(Landroid/content/Context;Ljava/lang/String;Ljava/lang/String;JLis;)J

    .line 162
    .line 163
    .line 164
    move-result-wide v2

    .line 165
    iput-wide v2, v1, Lns;->e:J

    .line 166
    .line 167
    sget-object p0, Lfm8;->a:Lfm8;
    :try_end_4
    .catchall {:try_start_4 .. :try_end_4} :catchall_0

    .line 168
    .line 169
    return-object p0

    .line 170
    :goto_4
    move p1, p2

    .line 171
    goto :goto_5

    .line 172
    :catchall_1
    move-exception v0

    .line 173
    move-object p0, v0

    .line 174
    goto :goto_4

    .line 175
    :cond_6
    :try_start_5
    const-string p0, "Required value was null."

    .line 176
    .line 177
    new-instance p1, Ljava/lang/IllegalArgumentException;

    .line 178
    .line 179
    invoke-direct {p1, p0}, Ljava/lang/IllegalArgumentException;-><init>(Ljava/lang/String;)V

    .line 180
    .line 181
    .line 182
    throw p1
    :try_end_5
    .catchall {:try_start_5 .. :try_end_5} :catchall_1

    .line 183
    :catchall_2
    move-exception v0

    .line 184
    move-object p0, v0

    .line 185
    move p1, v4

    .line 186
    :goto_5
    if-eqz p1, :cond_7

    .line 187
    .line 188
    invoke-static {p0}, Lih3;->Q(Ljava/lang/Throwable;)Ljava/lang/String;

    .line 189
    .line 190
    .line 191
    move-result-object p1

    .line 192
    const-string p2, "util.protection"

    .line 193
    .line 194
    invoke-static {p1, p2, v4}, Lgl7;->r0(Ljava/lang/CharSequence;Ljava/lang/CharSequence;Z)Z

    .line 195
    .line 196
    .line 197
    move-result p1

    .line 198
    if-nez p1, :cond_7

    .line 199
    .line 200
    invoke-virtual {p0}, Ljava/lang/Throwable;->printStackTrace()V

    .line 201
    .line 202
    .line 203
    :cond_7
    instance-of p1, p0, Ljava/lang/InterruptedException;

    .line 204
    .line 205
    if-nez p1, :cond_8

    .line 206
    .line 207
    instance-of p1, p0, Ljava/util/concurrent/CancellationException;

    .line 208
    .line 209
    if-nez p1, :cond_8

    .line 210
    .line 211
    new-instance p1, Lwi6;

    .line 212
    .line 213
    invoke-direct {p1, p0}, Lwi6;-><init>(Ljava/lang/Throwable;)V

    .line 214
    .line 215
    .line 216
    return-object p1

    .line 217
    :cond_8
    throw p0
.end method
