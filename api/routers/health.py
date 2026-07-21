from fastapi import APIRouter

router = APIRouter()

@router.get("/health")
@router.get("/health/")
@router.get("/api/health")
@router.get("/api/health/")
def health():
    return {"status": "ok"}
