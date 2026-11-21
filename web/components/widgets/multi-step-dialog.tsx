"use client"

import { useState } from "react"
import { toast } from "sonner"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { ChevronLeft } from "lucide-react"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"

export type FormField = {
  name: string
  label: string
  type?: string
  placeholder?: string
  maxLength?: number
  className?: string
  validate?: (
    value: string,
    allValues?: Record<string, string>
  ) => string | null
  formatter?: (value: string) => string
}

export type StepConfig = {
  title: string
  description: string
  fields: FormField[]
  actionText: string
  onSubmitAction?: (values: Record<string, string>) => Promise<boolean | void>
}

type MultiStepDialogProps = {
  className?: string
  triggerText: string
  steps: StepConfig[]
  onCompleteAction?: (finalValues: Record<string, string>) => Promise<void>
}

export function MultiStepDialog({
  className,
  triggerText,
  steps,
  onCompleteAction,
}: MultiStepDialogProps) {
  const [open, setOpen] = useState(false)
  const [currentStep, setCurrentStep] = useState(0)
  const [formValues, setFormValues] = useState<Record<string, string>>({})
  const [isLoading, setIsLoading] = useState(false)

  const handleChange = (name: string, value: string) => {
    setFormValues((prev) => ({ ...prev, [name]: value }))
  }

  const handleNext = async () => {
    const step = steps[currentStep]

    for (const field of step.fields) {
      const val = formValues[field.name] || ""
      if (field.validate) {
        // 🔥 修复：传入第二个参数 allValues
        const err = field.validate(val, formValues)
        if (err) {
          toast.error(err)
          return
        }
      }
    }

    setIsLoading(true)
    try {
      const success = await step.onSubmitAction?.(formValues)
      if (success === false) return

      if (currentStep === steps.length - 1) {
        await onCompleteAction?.(formValues)
        reset()
        return
      }

      setCurrentStep((prev) => prev + 1)
    } catch {
      toast.error("操作失败，请稍后重试")
    } finally {
      setIsLoading(false)
    }
  }

  const reset = () => {
    setOpen(false)
  }

  const handleOpenChange = (isOpen: boolean) => {
    if (!isOpen) {
      reset()
      setCurrentStep(0)
      setFormValues({})
    }
  }

  const step = steps[currentStep]

  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        className={cn(
          "ml-auto text-sm underline-offset-4 hover:underline",
          className
        )}
      >
        {triggerText}
      </button>

      <Dialog open={open} onOpenChange={handleOpenChange}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{step.title}</DialogTitle>
            <DialogDescription>{step.description}</DialogDescription>
            {/* 步骤进度指示器 */}
            <div className="flex items-center gap-1.5 pt-1">
              {steps.map((s, i) => (
                <div
                  key={i}
                  className={cn(
                    "h-1.5 flex-1 rounded-full transition-colors",
                    i <= currentStep ? "bg-primary" : "bg-muted"
                  )}
                />
              ))}
            </div>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            {step.fields.map((field) => (
              <div key={field.name} className="grid gap-3">
                <Label htmlFor={field.name}>{field.label}</Label>
                <Input
                  id={field.name}
                  type={field.type || "text"}
                  placeholder={field.placeholder}
                  maxLength={field.maxLength}
                  value={formValues[field.name] || ""}
                  onChange={(e) => {
                    let val = e.target.value
                    if (field.formatter) val = field.formatter(val)
                    handleChange(field.name, val)
                  }}
                  className={field.className}
                />
              </div>
            ))}
          </div>

          <DialogFooter>
            <div className="flex w-full gap-2">
              {currentStep > 0 && (
                <Button
                  variant="outline"
                  onClick={() => setCurrentStep((prev) => prev - 1)}
                  disabled={isLoading}
                >
                  <ChevronLeft className="mr-1 h-4 w-4" />
                  上一步
                </Button>
              )}
              <Button
                onClick={handleNext}
                disabled={isLoading}
                className="flex-1"
              >
                {isLoading ? "处理中..." : step.actionText}
              </Button>
            </div>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
