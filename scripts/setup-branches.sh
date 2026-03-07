#!/bin/bash
# GitHub Flow ブランチ設定スクリプト
# CLAUDE.md の開発ワークフローに合わせて、リモートリポジトリのブランチを整備する
#
# 前提: gh CLI がインストール済み・認証済みであること
#   gh auth login
#
# 使い方:
#   chmod +x scripts/setup-branches.sh
#   ./scripts/setup-branches.sh

set -euo pipefail

REPO="shunsuke-suzuki-zeroboard/sedori"
DEFAULT_BRANCH="main"
SOURCE_BRANCH="master"  # main が無い場合のフォールバック元

echo "=== GitHub Flow ブランチ設定 ==="
echo "リポジトリ: ${REPO}"
echo ""

# 1. リモートの最新状態を取得
echo "[1/5] リモートの最新状態を取得中..."
git fetch origin

# 2. main ブランチの作成（存在しない場合）
echo "[2/5] main ブランチの確認..."
if git show-ref --verify --quiet refs/remotes/origin/main; then
    echo "  → origin/main は既に存在します"
else
    echo "  → origin/main が存在しません。作成します..."

    # ローカルに main があればそれを使う、なければ master から作成
    if git show-ref --verify --quiet refs/heads/main; then
        echo "  → ローカルの main をプッシュします"
    elif git show-ref --verify --quiet refs/heads/${SOURCE_BRANCH}; then
        echo "  → ${SOURCE_BRANCH} から main を作成します"
        git branch main "${SOURCE_BRANCH}"
    else
        echo "エラー: main も ${SOURCE_BRANCH} も見つかりません" >&2
        exit 1
    fi

    git push -u origin main
    echo "  → origin/main を作成しました"
fi

# 3. デフォルトブランチを main に設定
echo "[3/5] デフォルトブランチを main に設定中..."
CURRENT_DEFAULT=$(gh repo view "${REPO}" --json defaultBranchRef --jq '.defaultBranchRef.name' 2>/dev/null || echo "unknown")
if [ "${CURRENT_DEFAULT}" = "${DEFAULT_BRANCH}" ]; then
    echo "  → デフォルトブランチは既に main です"
else
    gh repo edit "${REPO}" --default-branch "${DEFAULT_BRANCH}"
    echo "  → デフォルトブランチを main に変更しました（旧: ${CURRENT_DEFAULT}）"
fi

# 4. main ブランチの保護ルールを設定
echo "[4/5] main ブランチの保護ルールを設定中..."
# GitHub API 経由でブランチ保護を設定
# - 直接プッシュ禁止（PR 経由のみ）
# - PR には最低1件のレビュー承認が必要
gh api -X PUT "repos/${REPO}/branches/main/protection" \
    --input - <<'EOF' 2>/dev/null && echo "  → ブランチ保護ルールを設定しました" || echo "  → ブランチ保護の設定にはリポジトリの Admin 権限が必要です（スキップ）"
{
    "required_status_checks": null,
    "enforce_admins": false,
    "required_pull_request_reviews": {
        "required_approving_review_count": 1
    },
    "restrictions": null
}
EOF

# 5. 不要な master ブランチの案内
echo "[5/5] クリーンアップ..."
if git show-ref --verify --quiet refs/remotes/origin/master; then
    echo "  → origin/master が残っています。不要であれば以下で削除できます:"
    echo "    git push origin --delete master"
fi
if git show-ref --verify --quiet refs/heads/master; then
    echo "  → ローカルの master が残っています。不要であれば以下で削除できます:"
    echo "    git branch -d master"
fi

echo ""
echo "=== 設定完了 ==="
echo ""
echo "ブランチ運用ルール（GitHub Flow）:"
echo "  - main: 常にデプロイ可能な状態を維持（直接コミット禁止）"
echo "  - feature/*: 機能開発ブランチ（main から分岐、PR 経由でマージ）"
echo "  - fix/*: バグ修正ブランチ"
echo "  - docs/*: ドキュメント変更ブランチ"
echo ""
echo "作業の流れ:"
echo "  git checkout main && git pull origin main"
echo "  git checkout -b feature/<機能名>"
echo "  # ... 実装 ..."
echo "  git push -u origin feature/<機能名>"
echo "  # GitHub で PR 作成 → レビュー → Squash and merge"
