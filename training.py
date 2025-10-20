# training.py
import os
import random
import torch
import torch.nn as nn
import torch.optim as optim
from tqdm import tqdm

from data_loader import get_sequence_loaders
from network import DrowsinessNet

# --------------------
# Конфиг
# --------------------
DATA_DIR = "processed_dataset"
IMG_SIZE = 112

#Для первого запуска сделай поменьше
BATCH_SIZE = 8
EPOCHS = 5

LR = 2e-3
WEIGHT_DECAY = 1e-4
NUM_WORKERS = 0
CHECKPOINT_DIR = "checkpoints"
os.makedirs(CHECKPOINT_DIR, exist_ok=True)

# Ограничение числа батчей (ускоряет пробный прогон).
# Поставь None, когда захочешь обучать полноценно.
LIMIT_TRAIN_BATCHES = None
LIMIT_VAL_BATCHES   = None

SEED = 42

# --------------------
# Утилиты
# --------------------
def set_seed(seed=SEED):
    random.seed(seed)
    torch.manual_seed(seed)
    torch.cuda.manual_seed_all(seed)

def bin_metrics(pred_prob, y_true, thr=0.5):
    """accuracy и F1 для бинарной классификации"""
    y_pred = (pred_prob >= thr).long()
    acc = (y_pred == y_true.long()).float().mean().item()
    tp = ((y_pred == 1) & (y_true == 1)).sum().item()
    fp = ((y_pred == 1) & (y_true == 0)).sum().item()
    fn = ((y_pred == 0) & (y_true == 1)).sum().item()
    precision = tp / (tp + fp + 1e-9)
    recall    = tp / (tp + fn + 1e-9)
    f1        = 2 * precision * recall / (precision + recall + 1e-9)
    return acc, f1

# --------------------
# Тренировка / Валидация
# --------------------
def train_one_epoch(model, loader, optimizer, scaler, criterion, device, limit_batches=None):
    model.train()
    total_loss = total_acc = total_f1 = 0.0
    n_batches = 0

    it = enumerate(loader)
    for i, (eyes, mouths, poses, labels) in tqdm(
        it,
        total=(limit_batches or len(loader)),
        desc="train",
        leave=False
    ):
        if limit_batches and i >= limit_batches:
            break

        eyes   = eyes.to(device)
        mouths = mouths.to(device)
        poses  = poses.to(device)
        labels = labels.to(device).float()

        optimizer.zero_grad(set_to_none=True)
        if scaler is not None:
            with torch.autocast(device_type="cuda", dtype=torch.float16):
                probs = model(eyes, mouths, poses)   # [B], sigmoid внутри сети
                loss = criterion(probs, labels)
            scaler.scale(loss).backward()
            scaler.step(optimizer)
            scaler.update()
        else:
            probs = model(eyes, mouths, poses)
            loss  = criterion(probs, labels)
            loss.backward()
            optimizer.step()

        acc, f1 = bin_metrics(probs.detach(), labels.detach())
        total_loss += loss.item()
        total_acc  += acc
        total_f1   += f1
        n_batches  += 1

    return total_loss / max(1, n_batches), total_acc / max(1, n_batches), total_f1 / max(1, n_batches)


@torch.no_grad()
def evaluate(model, loader, criterion, device, limit_batches=None):
    model.eval()
    total_loss = total_acc = total_f1 = 0.0
    n_batches = 0

    it = enumerate(loader)
    for i, (eyes, mouths, poses, labels) in tqdm(
        it,
        total=(limit_batches or len(loader)),
        desc="val",
        leave=False
    ):
        if limit_batches and i >= limit_batches:
            break

        eyes   = eyes.to(device)
        mouths = mouths.to(device)
        poses  = poses.to(device)
        labels = labels.to(device).float()

        probs = model(eyes, mouths, poses)
        loss  = criterion(probs, labels)
        acc, f1 = bin_metrics(probs, labels)

        total_loss += loss.item()
        total_acc  += acc
        total_f1   += f1
        n_batches  += 1

    return total_loss / max(1, n_batches), total_acc / max(1, n_batches), total_f1 / max(1, n_batches)

# --------------------
# Главный запуск
# --------------------
if __name__ == "__main__":
    set_seed()

    device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
    print("Device:", device)
    if device.type == "cuda":
        torch.backends.cudnn.benchmark = True

    # Dataloaders
    train_loader, val_loader, test_loader, classes = get_sequence_loaders(
        root_dir=DATA_DIR,
        batch_size=BATCH_SIZE,
        img_size=IMG_SIZE,
        num_workers=NUM_WORKERS
    )
    print("Классы:", classes)
    print(f"batches -> train: {len(train_loader)} | val: {len(val_loader)} | test: {len(test_loader)}")

    # Модель
    model = DrowsinessNet().to(device)

    # Лосс/оптимизатор/шедулер
    criterion = nn.BCELoss()
    optimizer = optim.AdamW(model.parameters(), lr=LR, weight_decay=1e-4)
    scheduler = optim.lr_scheduler.ReduceLROnPlateau(optimizer, mode="min", factor=0.5, patience=2)

    # AMP scaler (без депрекейта)
    scaler = torch.amp.GradScaler(device.type) if device.type == "cuda" else None

    best_val_f1 = -1.0
    best_path = os.path.join(CHECKPOINT_DIR, "best_model.pth")

    for epoch in range(1, EPOCHS + 1):
        print(f"\nEpoch {epoch}/{EPOCHS}")

        train_loss, train_acc, train_f1 = train_one_epoch(
            model, train_loader, optimizer, scaler, criterion, device,
            limit_batches=LIMIT_TRAIN_BATCHES
        )
        val_loss, val_acc, val_f1 = evaluate(
            model, val_loader, criterion, device,
            limit_batches=LIMIT_VAL_BATCHES
        )

        scheduler.step(val_loss)

        print(f"  train: loss {train_loss:.4f}, acc {train_acc:.3f}, f1 {train_f1:.3f}")
        print(f"  val:   loss {val_loss:.4f}, acc {val_acc:.3f}, f1 {val_f1:.3f}")

        if val_f1 > best_val_f1:
            best_val_f1 = val_f1
            torch.save(model.state_dict(), best_path)
            print(f"saved best checkpoint -> {best_path} (F1={best_val_f1:.3f})")

    # --- Финальный тест ---
    print("\n[TEST]")
    model.load_state_dict(torch.load(best_path, map_location=device))
    test_loss, test_acc, test_f1 = evaluate(model, test_loader, criterion, device)
    print(f"  test:  loss {test_loss:.4f}, acc {test_acc:.3f}, f1 {test_f1:.3f}")

    final_path = os.path.join(CHECKPOINT_DIR, "final_model.pth")
    torch.save(model.state_dict(), final_path)
    print(f"final model saved to: {final_path}")

    print("\nTip: когда убедишься, что всё ок — убери LIMIT_* и подними BATCH_SIZE/EPOCHS для нормального обучения.")
