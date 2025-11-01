import os, random, torch, torch.nn as nn, torch.optim as optim
from tqdm import tqdm
from data_loader import get_loaders
from network import TinyCNN

# ==== настройки ====
DATA_DIR = r"archive/Driver Drowsiness Dataset (DDD)"
IMG_SIZE = 160
BATCH = 32
EPOCHS = 10
LR = 2e-3
WD = 1e-4
NUM_WORKERS = 0
CHECKPOINT_DIR = "checkpoints"
os.makedirs(CHECKPOINT_DIR, exist_ok=True)
SEED = 42


def set_seed(s=SEED):
    random.seed(s); torch.manual_seed(s); torch.cuda.manual_seed_all(s)


def epoch_loop(model, loader, opt, crit, device, train=True):
    if train: model.train()
    else: model.eval()
    total_loss=0; correct=0; n=0
    pbar = tqdm(loader, leave=False, desc="train" if train else "val")
    with torch.set_grad_enabled(train):
        for x, y in pbar:
            x, y = x.to(device), y.to(device).float()
            if train: opt.zero_grad(set_to_none=True)
            logits = model(x)
            loss = crit(logits, y)
            if train:
                loss.backward(); opt.step()
            prob = torch.sigmoid(logits)
            pred = (prob>=0.5).long()
            correct += (pred==y.long()).sum().item()
            total_loss += loss.item()*x.size(0)
            n += x.size(0)
            pbar.set_postfix(loss=f"{total_loss/n:.4f}", acc=f"{correct/n:.3f}")
    return total_loss/n, correct/n


if __name__ == "__main__":
    set_seed()
    device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
    print("Device:", device)

    train_loader, val_loader, test_loader, classes = get_loaders(
        DATA_DIR, IMG_SIZE, BATCH, NUM_WORKERS, train_ratio=0.8, val_ratio=0.1
    )
    print("Classes:", classes)

    model = TinyCNN(num_classes=1).to(device)
    crit = nn.BCEWithLogitsLoss()
    opt = optim.AdamW(model.parameters(), lr=LR, weight_decay=WD)
    sched = optim.lr_scheduler.ReduceLROnPlateau(opt, mode="min", factor=0.5, patience=2)

    best_val = 1e9; best_path = os.path.join(CHECKPOINT_DIR, "best_simple.pth")

    for epoch in range(1, EPOCHS+1):
        print(f"\nEpoch {epoch}/{EPOCHS}")
        tr_loss, tr_acc = epoch_loop(model, train_loader, opt, crit, device, train=True)
        va_loss, va_acc = epoch_loop(model, val_loader,   opt, crit, device, train=False)
        sched.step(va_loss)
        print(f"train: loss {tr_loss:.4f}, acc {tr_acc:.3f} | val: loss {va_loss:.4f}, acc {va_acc:.3f}")
        if va_loss < best_val:
            best_val = va_loss
            torch.save(model.state_dict(), best_path)
            print("  ✔ saved:", best_path)

    # тест
    model.load_state_dict(torch.load(best_path, map_location=device))
    te_loss, te_acc = epoch_loop(model, test_loader, opt, crit, device, train=False)
    print(f"\n[TEST] loss {te_loss:.4f}, acc {te_acc:.3f}")
    torch.save(model.state_dict(), os.path.join(CHECKPOINT_DIR, "final_simple.pth"))
    print("final saved -> checkpoints/final_simple.pth")