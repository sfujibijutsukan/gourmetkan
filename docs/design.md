# 研究室ご飯屋共有アプリ 設計書

## 1. システム概要

研究室メンバーでおすすめの飲食店情報を共有・蓄積するための Web アプリケーション。日々のランチや飲み会の場所決めのコストを下げ、メンバー間のコミュニケーションを促進する。

### 1.1. 技術スタック

- **バックエンド & フロントエンド:** Go 言語（標準ライブラリ `net/http`, `html/template` を基本とし、追加依存は最小）
- **データベース:** SQLite（`github.com/mattn/go-sqlite3`）
- **認証:** GitHub OAuth 2.0
- **外部サービス:** Google Maps（地図表示・経路案内 URL 生成、URL からの位置情報抽出）

### 1.2. システムアーキテクチャ

```
+-------------------------------------------------------------+
|                     Client (Browser/Mobile)                 |
|                                                             |
|  +-----------------------+         +---------------------+  |
|  |   UI (HTML/JS/CSS)    | ------> |   Google Maps App   |  |
|  | (Template Rendered)   | (Link)  | (Route Navigation)  |  |
|  +-----------+-----------+         +---------------------+  |
|              |                                              |
+--------------|----------------------------------------------+
               | HTTP Requests (GET/POST)
               | Cookie (Session ID, Selected base_id)
               v
+-------------------------------------------------------------+
|                      Web Server (Go)                        |
|                                                             |
|  +-----------------+        +----------------------------+  |
|  |                 |        |      Business Logic        |  |
|  |   HTTP Router   |        |                            |  |
|  |                 |        | - Distance Calculation     |  |
|  +-------+---------+        |   (Haversine formula)      |  |
|          |                  |                            |  |
|          v                  | - Map URL Parsing (Regexp) |  |
|  +-----------------+        |   (Extract Lat/Lng)        |  |
|  |                 | <----->|                            |  |
|  |  Controllers    |        +-------------+--------------+  |
|  |                 |                      |                 |
|  +-------+---------+                      |                 |
|          | SQL Queries                    | OAuth 2.0 API   |
+----------|--------------------------------|-----------------+
           |                                |
           v                                v
+-----------------------+        +----------------------+
|       Database        |        |   External Service   |
|       (SQLite)        |        |                      |
|                       |        |  +----------------+  |
|  - users              |        |  |  GitHub API    |  |
|  - bases              |        |  | (OAuth Login)  |  |
|  - restaurants        |        |  +----------------+  |
|  - reviews            |        |                      |
+-----------------------+        +----------------------+
```

---

## 2. 機能一覧

| カテゴリ | 機能名 | 説明 |
| :--- | :--- | :--- |
| **認証** | GitHubログイン | GitHubアカウントを用いたOAuth2.0ログイン・ログアウト機能。 |
| **拠点** | 拠点切り替え | 「〇〇キャンパス」「〇〇研究所」など、基準となる拠点を画面上で切り替える機能。 |
| **店舗** | 店舗一覧・距離ソート | 登録された店舗リストを表示。**現在選択している拠点からの距離が近い順**にソートして表示。 |
|  | 店舗詳細表示 | 店舗の基本情報、地図、口コミ一覧（アプリ内でメンバーが投稿したもののみ）、選択中拠点からの距離を表示。 |
|  | 店舗登録 | 店名、説明、Google Maps の URL 等を入力し店舗を登録。**URL から緯度経度を抽出、または直接入力**して保存する。 |
|  | ランダム提案 | 登録された店舗の中からランダムに 1 件を抽出して提案する機能。 |
| **口コミ** | 口コミ投稿 | 5段階評価（星）とコメントを投稿。 |
| **便利機能** | 経路検索リンク | **選択中の拠点**から店舗までの経路（徒歩/電車/車）を Google Maps 等で開くリンクを生成。 |

---

## 3. 非機能要件

### 3.1. セキュリティ

- OAuth state 検証、CSRF 対策（POST は CSRF トークン必須）
- セッション Cookie: HttpOnly, SameSite=Lax, Secure（HTTPS 運用時）
- 位置情報入力のバリデーション（緯度: -90〜90, 経度: -180〜180）
- 認可: 店舗登録・口コミ投稿はログイン必須。未ログイン時はログインページへリダイレクト。

### 3.2. パフォーマンス

- 店舗数数千件規模を想定
- 一覧画面の距離計算は Go 側で実施
- 口コミ取得はページネーション（デフォルト 20 件）

### 3.3. 可用性/運用

- SQLite ファイルのバックアップ（週次 + 手動）
- ログ出力（アクセスログ + アプリログ）

### 3.4. エラーハンドリング

- ユーザー向け: 400/422 は入力エラーとしてフォームに表示、401/403 はログインまたは権限不足表示、500 は汎用エラー画面。
- 監視向け: 5xx は必ずアプリログにスタック/原因を記録。

---

## 4. データベース設計（SQLite スキーマ）

### 4.1. テーブル定義

#### 4.1.1. users（ユーザーテーブル）

| カラム名 | 型 | 制約 | 説明 |
| :--- | :--- | :--- | :--- |
| id | INTEGER | PRIMARY KEY, AUTOINCREMENT | 内部ユーザーID |
| github_id | TEXT | UNIQUE, NOT NULL | GitHub のユーザーID |
| username | TEXT | NOT NULL | GitHub のユーザー名 |
| avatar_url | TEXT |  | GitHub のアイコン画像URL |
| created_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | 登録日時 |
| updated_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | 更新日時 |

#### 4.1.2. bases（拠点テーブル）

| カラム名 | 型 | 制約 | 説明 |
| :--- | :--- | :--- | :--- |
| id | INTEGER | PRIMARY KEY, AUTOINCREMENT | 拠点ID |
| name | TEXT | NOT NULL | 拠点名（例: 〇〇キャンパス） |
| latitude | REAL | NOT NULL | 拠点の緯度 |
| longitude | REAL | NOT NULL | 拠点の経度 |
| created_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | 登録日時 |
| updated_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | 更新日時 |

#### 4.1.3. restaurants（店舗テーブル）

| カラム名 | 型 | 制約 | 説明 |
| :--- | :--- | :--- | :--- |
| id | INTEGER | PRIMARY KEY, AUTOINCREMENT | 店舗ID |
| name | TEXT | NOT NULL | 店舗名 |
| description | TEXT |  | 簡単な説明・特徴 |
| latitude | REAL | NOT NULL | 緯度（経路計算・地図表示用） |
| longitude | REAL | NOT NULL | 経度（経路計算・地図表示用） |
| address | TEXT |  | 住所（自由入力） |
| maps_url | TEXT |  | Google Maps 共有 URL |
| created_by | INTEGER | NOT NULL | 登録したユーザーのID |
| created_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | 登録日時 |
| updated_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | 更新日時 |

#### 4.1.4. reviews（口コミテーブル）

| カラム名 | 型 | 制約 | 説明 |
| :--- | :--- | :--- | :--- |
| id | INTEGER | PRIMARY KEY, AUTOINCREMENT | 口コミID |
| restaurant_id | INTEGER | NOT NULL | 対象店舗ID |
| user_id | INTEGER | NOT NULL | 投稿ユーザーID |
| rating | INTEGER | CHECK(rating >= 1 AND rating <= 5) | 評価（1〜5） |
| comment | TEXT | NOT NULL | 口コミ本文 |
| created_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | 投稿日時 |
| updated_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | 更新日時 |

#### 4.1.5. sessions（セッションテーブル）

| カラム名 | 型 | 制約 | 説明 |
| :--- | :--- | :--- | :--- |
| id | TEXT | PRIMARY KEY | セッションID（ランダム） |
| user_id | INTEGER | NOT NULL | users.id |
| csrf_token | TEXT | NOT NULL | CSRF トークン |
| created_at | DATETIME | DEFAULT CURRENT_TIMESTAMP | 作成日時 |
| expires_at | DATETIME | NOT NULL | 失効日時 |

### 4.2. 外部キー制約

- `restaurants.created_by` → `users.id`（ON DELETE RESTRICT）
- `reviews.restaurant_id` → `restaurants.id`（ON DELETE CASCADE）
- `reviews.user_id` → `users.id`（ON DELETE RESTRICT）
- `sessions.user_id` → `users.id`（ON DELETE CASCADE）

### 4.3. インデックス

- `users.github_id`（UNIQUE）
- `restaurants.created_by`
- `restaurants.latitude`, `restaurants.longitude`（距離計算前提の簡易インデックス）
- `reviews.restaurant_id`
- `reviews.user_id`
- `sessions.user_id`
- `sessions.expires_at`

### 4.4. 口コミ重複ポリシー

- 同一ユーザーの同一店舗への複数投稿は許可する（履歴として残す）。
- 将来の編集/削除は Phase 2 とし、現行は投稿のみ。

---

## 5. 認証設計

### 5.1. GitHub OAuth フロー

1. `/auth/github/login`
   - `state` を生成しサーバー側に保存（セッション）
   - GitHub OAuth 認証 URL へリダイレクト
2. `/auth/github/callback`
   - `state` を検証
   - 認可コードからアクセストークンを取得
   - GitHub API でユーザー情報取得
   - `users` テーブルに upsert（github_id で一意）
   - セッション作成 → Cookie へセッション ID 発行

### 5.2. セッション設計

- サーバー側に `sessions` テーブルを持つ（DB 管理）
- Cookie に `session_id` を保存
- セッション期限: 14 日
- Cookie 属性: `HttpOnly`, `SameSite=Lax`, `Secure`（HTTPS のみ）
- CSRF: `sessions.csrf_token` をフォームに埋め込み、POST 時に一致検証

### 5.3. ログアウト

- `/auth/logout` で sessions を削除し Cookie を失効させる。

---

## 6. 画面/テンプレート設計

### 6.1. 共通レイアウト

- ヘッダー: 拠点選択ドロップダウン / ログイン状態
- フッター: アプリ名、簡易説明

### 6.2. 画面一覧

1. **トップ（/）**
   - 店舗一覧（距離順）
   - フィルタ: 拠点選択
   - ボタン: ランダム提案
2. **店舗詳細（/restaurants/{id}）**
   - 店舗情報 + 地図リンク
   - 口コミ一覧 + 投稿フォーム
3. **店舗登録（/restaurants/new）**
   - 店舗名/説明/地図 URL/緯度経度の入力

### 6.3. エラー画面

- 400/422: 入力エラーをフォーム上に表示
- 401/403: ログイン案内ページ
- 404: Not Found
- 500: 汎用エラー画面

---

## 7. API/エンドポイント設計

| HTTPメソッド | パス | 説明 | 認証 | 主要パラメータ |
| :--- | :--- | :--- | :--- | :--- |
| GET | / | 店舗一覧（距離順） | 任意 | base_id |
| GET | /auth/github/login | GitHub OAuth 認証画面へリダイレクト | なし | なし |
| GET | /auth/github/callback | GitHub コールバック処理 | なし | code, state |
| POST | /auth/logout | ログアウト | 必須 | なし |
| POST | /bases/select | 拠点変更 | 任意 | base_id |
| GET | /restaurants/new | 店舗登録フォーム | 必須 | なし |
| POST | /restaurants | 店舗登録 | 必須 | name, description, maps_url, latitude, longitude, address |
| GET | /restaurants/{id} | 店舗詳細 | 任意 | なし |
| POST | /restaurants/{id}/reviews | 口コミ投稿 | 必須 | rating, comment |
| GET | /random | ランダム提案 | 任意 | radius_km (任意) |

---

## 8. 主要処理フロー

### 8.1. 店舗一覧表示

1. Cookie から base_id を取得（なければ default base を DB から取得）
2. restaurants を DB から取得
3. Go 側で距離計算（Haversine）
4. 距離順にソートして表示

### 8.1.1. 拠点選択の取り扱い

- base_id は Cookie に保存するが、毎回 DB で存在チェックする。
- 不正な base_id の場合は default base を採用し、Cookie を上書き。
- default base は `bases` の最小 id とする（固定値ではない）。

### 8.2. 店舗登録

1. 入力値のバリデーション
2. maps_url がある場合は URL 展開と緯度経度抽出を試行
3. 抽出失敗時は直接入力値を採用（必須ではない）
4. restaurants に INSERT

### 8.3. 口コミ投稿

1. 認証チェック
2. rating/comment のバリデーション
3. reviews に INSERT

### 8.4. ランダム提案

1. base_id を取得
2. 近距離店舗のみ抽出（radius_km 既定値 2km）
3. 1件をランダム選択してリダイレクト

### 8.5. ルーティングの認可

- `/restaurants/new`, `POST /restaurants`, `POST /restaurants/{id}/reviews`, `POST /auth/logout` はログイン必須。
- 未ログイン時は 302 で `/auth/github/login` に遷移。

---

## 9. バリデーション仕様

| 項目 | 制約 |
| :--- | :--- |
| 店舗名 | 必須、1〜100文字 |
| 説明 | 任意、0〜500文字 |
| maps_url | 任意、URL 形式 |
| 緯度 | 任意、-90〜90 |
| 経度 | 任意、-180〜180 |
| rating | 必須、1〜5 |
| comment | 必須、1〜1000文字 |

---

## 10. 距離表示仕様

- 1000m 未満: `xxx m`
- 1000m 以上: `x.x km`（小数 1 桁）

---

## 11. 実装の補足（Go）

- 距離計算: Haversine 公式
- URL 抽出: 正規表現 `@(-?\d+\.\d+),(-?\d+\.\d+)` を優先し、失敗時は `q=lat,lng` パターンを試行
- 短縮 URL 展開: HTTP HEAD/GET を最大 3 リダイレクト、2 秒タイムアウト
- Google Maps URL なしで緯度経度の直接入力も可（片方欠けは不可）

## 12. デプロイ/実行環境

- SQLite ファイルは `./data/app.db` に配置（起動時に存在しなければ作成）
- `./data` は書き込み権限が必要
- バックアップは `./backup` に日付付きでコピー

---

## 13. 今後の拡張アイデア（Phase 2 以降）

- **Slack/Discord 通知:** 新しいお店が登録されたり、口コミが書かれたら Webhook 経由で通知。
- **現在地からの検索:** スマホの GPS（HTML5 Geolocation API）を利用して、「今の自分の場所」を一時的な拠点として距離ソートする機能。
